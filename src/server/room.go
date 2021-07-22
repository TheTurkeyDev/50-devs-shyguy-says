package server

import (
	"log"
	"shyguy-says/src/common"
	"time"

	"nhooyr.io/websocket"
)

type Room struct {
	ticking  bool
	roomName string
	password string
	clients  []*Client
}

func (r *Room) sendMessageToAll(msg interface{}) {
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

func (r *Room) sendMessageToAllExcluding(msg interface{}, exclude *Client) {
	for _, client := range r.clients {
		if client == exclude {
			continue
		}

		err := client.sendMessage(msg)
		if err != nil {
			log.Printf("error: %v", err)
			// TODO: Probably be a bit mroe graceful than this
			client.con.Close(websocket.StatusInternalError, err.Error())
			// delete(r.clients, client)
		}
	}
}

func (r *Room) runGame() {
	r.ticking = true
	game := Game{
		room: r,
	}
	game.init()

	for r.ticking {
		game.tick()
		time.Sleep(time.Millisecond * 50)
	}
}

func (r *Room) stopGame() {
	r.ticking = false
}

func (r *Room) onClientLeave(leavingClient *Client) {
	rc := r.clients
	for i, c := range rc {
		if c == leavingClient {
			rc = append(rc[:i], rc[i+1:]...)
			break
		}
	}
	r.clients = rc

	if len(r.clients) == 1 {
		r.stopGame()
		r.sendMessageToAll(common.GameOverPacket{
			GenericMessage: common.GenericMessage{
				Type: common.GameOver,
			},
			Data: common.GameOverData{
				Winner: r.clients[0].player,
			},
		})
	}
}
