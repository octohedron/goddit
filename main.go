package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"log"
	"net/http"
	"time"
)

type User struct {
	Name             string `bson:"name" json:"name"`
	Level            string `bson:"level" json:"level"`
	Active           string `bson:"active" json:"active"`
	Activation_token string `bson:"activation_token" json:"activation_token"`
	Created_at       string `bson:"created_at" json:"created_at"`
}

type Chatroom struct {
	Id         bson.ObjectId   `bson:"_id,omitempty" json:"_id,omitempty" inline`
	Name       string          `bson:"name" json:"name"`
	Level      string          `bson:"level" json:"level"`
	Active     string          `bson:"active" json:"active"`
	Created_at string          `bson:"created_at" json:"created_at"`
	Messages   []bson.ObjectId `bson:"messages,omitempty" json:"messages" inline`
}

type Message struct {
	MessageId    bson.ObjectId `bson:"_id,omitempty" json:"_id,omitempty" inline`
	Text         string        `bson:"text" json:"text"`
	UserName     string        `bson:"name" json:"name"`
	ChatRoomName string        `bson:"room_name" json:"room_name"`
	ChatRoomId   bson.ObjectId `bson:"chatRoomId,omitempty" json:"chatRoomId,omitempty"`
}

var addr = flag.String("addr", ":9000", "http service address")

const project_root = "/home/vagrant/GO/chat"

/**
 * Chat channel
 */
func serveIndex(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	// serve
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	t, _ := template.ParseFiles(project_root + "/index.html")
	t.Execute(w, r)
}

/**
 * Load the previous messages from this channel from the database
 */
func serveChannelHistory(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	vars := mux.Vars(r)
	// connect to the database
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	// close the session when done
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("views").C("chatrooms")
	m := session.DB("views").C("messages")
	var room Chatroom
	// find the chatroom at this request
	err = c.Find(bson.M{"name": vars["channel"]}).One(&room)
	if err != nil { // channel not found
		log.Printf("Creating new channel %s ...", vars["channel"])
		// create new channel
		room.Id = bson.NewObjectId()
		room.Name = vars["channel"]
		room.Level = "0"
		room.Active = "true"
		err := c.Insert(room)
		if err != nil {
			log.Println(err)
		} else {
			// new welcome message for the room
			welcomeMessage := Message{
				MessageId:    bson.NewObjectId(),
				Text:         "Welcome to this new channel",
				ChatRoomName: vars["channel"],
				UserName:     "Moderator",
				ChatRoomId:   room.Id,
			}
			room.Messages = append(room.Messages, welcomeMessage.MessageId)
			// insert the new welcome message into the messages
			// collection, with this chatroom id and the user id
			err = m.Insert(welcomeMessage)
			if err != nil {
				panic(err) // error inserting
			}
		}
	} else { // channel found
		log.Println("Channel found!")
	}
	// initialize a slice of size messageAmount to store the messages
	var messageSlice []Message
	// find all the messages in this chatroom
	err = m.Find(bson.M{"chatRoomId": room.Id}).All(&messageSlice)
	// get json
	js, err := json.Marshal(messageSlice)
	if err != nil {
		panic(err)
	}
	// serve
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func main() {
	flag.Parse()
	r := mux.NewRouter()
	hub := newHub()
	go hub.run()
	r.HandleFunc("/", serveIndex)
	// fetch this payload when loading the chat client from web/mobile
	r.HandleFunc("/history/{channel}", serveChannelHistory)
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Serving websocket")
		serveWs(hub, w, r)
	})
	srv := &http.Server{
		Handler: r,
		Addr:    *addr,
		// Enforcing timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
