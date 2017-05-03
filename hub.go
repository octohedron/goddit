package main

import (
	"encoding/json"
	"log"
)

// hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			msg := Message{}
			err := json.Unmarshal(message, &msg)
			if err != nil {
				panic(err)
			}
			// only send message to clients that belong to this hub
			for client := range h.clients {
				// if client belongs to (message.room)
				// only send the message to the people in the room
				if client.room == msg.ChatRoomName {
					log.Printf("Client room: %s message room: %s \n",
						client.room, msg.ChatRoomName)
					select {
					// send only to the client if the message belongs to this room.
					case client.send <- message:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				} else {
					log.Printf("Client room: doesn't match %s message room: %s \n",
						client.room, msg.ChatRoomName)
				}
			}
		}
	}
}
