package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"shyguy-says/src/common"
	"strings"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var (
	clients = make(map[*websocket.Conn]string) // connected clients
	rooms   = make(map[string]Room)
)

func main() {
	log.SetFlags(0)

	fs := http.FileServer(http.Dir("./dist"))
	http.Handle("/", fs)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Print("Connection!")

	// Make sure we close the connection when the function returns
	defer ws.Close(websocket.StatusInternalError, "Closing!")

	// Register our new client
	clients[ws] = ""

	for {
		// Read in a new message as JSON and map it to a Message object
		_, bytes, err := ws.Read(r.Context())
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}

		var message common.GenericMessage
		if err := json.Unmarshal(bytes, &message); err != nil {
			fmt.Println("error:", err)
		}

		switch message.Type {
		case common.JoinRoom:
			var joinMsg common.JoinRoomMessage
			if err := json.Unmarshal(bytes, &joinMsg); err != nil {
				fmt.Println("error:", err)
			}

			result := common.JoinRoomResultMessage{
				GenericMessage: common.GenericMessage{
					Type: common.JoinRoomResult,
				},
				Data: common.JoinResult{
					Room:    joinMsg.Data,
					Valid:   true,
					Message: "",
				},
			}

			if joinMsg.Data.Name != "test" || joinMsg.Data.Password != "test" {
				result.Data.Valid = false
				result.Data.Message = "Room name or password invalid!"
			}

			sendMessage(ws, result)
		case common.CreateRoom:
			var joinMsg common.JoinRoomMessage
			if err := json.Unmarshal(bytes, &joinMsg); err != nil {
				fmt.Println("error:", err)
			}

			roomName := joinMsg.Data.Name

			result := common.JoinRoomResultMessage{
				GenericMessage: common.GenericMessage{
					Type: common.CreateRoomResult,
				},
				Data: common.JoinResult{
					Room:    joinMsg.Data,
					Valid:   true,
					Message: "",
				},
			}

			if len(strings.TrimSpace(roomName)) == 0 {
				result.Data.Valid = false
				result.Data.Message = "Room name cannot be empty!"
			} else {
				room := Room{
					roomName: roomName,
					password: joinMsg.Data.Password,
					clients: []*Client{{
						con: ws,
						id:  0,
					}},
				}
				rooms[roomName] = room
				clients[ws] = roomName
			}

			sendMessage(ws, result)
		}
	}
}

func sendMessage(client *websocket.Conn, msg interface{}) {
	err := wsjson.Write(context.Background(), client, msg)
	if err != nil {
		log.Printf("error: %v", err)
		client.Close(websocket.StatusInternalError, err.Error())
		delete(clients, client)
	}
}
