package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		msg := Message{}
		err = json.Unmarshal(message, &msg)
		// log.Printf("%+v\n", msg)
		// log.Println(msg.UserName)
		// log.Println(msg.Text)
		if err != nil {
			panic(err)
		}
		// log.Println(string(message[:]))
		saveMessage(&msg)
		// make hub broadcast the message
		// c.hub.broadcast <- message
		c.hub.broadcast <- []byte(msg.UserName + ": " + msg.Text)
	}
}

func saveMessage(message *Message) {
	message.MessageId = bson.NewObjectId()
	// save message
	// connect to the database
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	// close the session when done
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	// select the collections to work with
	c := session.DB("views").C("chatrooms")
	m := session.DB("views").C("messages")
	var room Chatroom
	// find the chatroom at this request
	// feature missing, get the chatroom name or id, hardcoding "devs"
	// channel for now
	err = c.Find(bson.M{"name": message.ChatRoomName}).One(&room)
	if err != nil { // channel not found
		fmt.Println("Channel not found, creating new channel...")
		// create new channel
		room.Name = message.ChatRoomName
		room.Level = "0"
		room.Active = "true"
		room.Id = bson.NewObjectId()
		err := c.Insert(room)
		if err != nil {
			log.Println(err)
		} else {
			log.Println("No errors creating the room, appending room id")
			room.Messages = append(room.Messages, message.MessageId)
			log.Println(room.Messages)
		}
	} else { // channel found
		fmt.Println("Channel found!")
	}
	// construct the new message
	message.ChatRoomId = room.Id
	// insert the message into the messages collection, with this chatroom
	// and the user id
	err = m.Insert(message)
	if err != nil {
		panic(err) // error inserting
	}
	var messageSlice []Message
	var bsonMessageSlice []bson.ObjectId
	// find all the messages that have this room as chatRoomId
	err = m.Find(bson.M{"chatRoomId": room.Id}).All(&messageSlice)
	if err != nil {
		panic(err)
	}
	if len(messageSlice) > 0 {
		fmt.Println("Found messages in the chatroom")
		if err != nil {
			log.Println(err)
		}
		// if there is no messages it won't enter the loop
		for i := 0; i < len(messageSlice); i++ {
			bsonMessageSlice = append(bsonMessageSlice, messageSlice[i].MessageId)
		}
	} else {
		fmt.Println("Something wrong, couldn't find messages in the chatroom")
	}
	// append the new message
	bsonMessageSlice = append(bsonMessageSlice, message.MessageId)
	// update the room with the new messsage
	// Update the chatroom with this room's id, adding the last message
	err = c.Update(bson.M{"_id": room.Id},
		bson.M{"$set": bson.M{"messages": bsonMessageSlice}})
	if err != nil {
		panic(err)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client
	go client.writePump()
	client.readPump()
}
