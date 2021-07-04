package main

import (
	"log"

	"nhooyr.io/websocket"
)

type Room struct {
	roomName string
	password string
	clients  []*Client
}

func (r *Room) sendMessageToAll(client *websocket.Conn, msg interface{}) {
	for _, client := range r.clients {
		err := client.sendMessage(msg)
		if err != nil {
			log.Printf("error: %v", err)
			// TODO: Probably be a bit mroe graceful than this
			client.con.Close(websocket.StatusInternalError, err.Error())
			// delete(r.clients, client)
		}
	}
}
