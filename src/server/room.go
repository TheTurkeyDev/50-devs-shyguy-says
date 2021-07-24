package server

import (
	"log"
	"shyguy-says/src/common"
	"time"

	"nhooyr.io/websocket"
)

type Room struct {
	Ticking  bool
	RoomName string
	Password string
	Clients  []*Client
	Game     *Game
}

func (r *Room) sendMessageToAll(msg interface{}) {
	for _, client := range r.Clients {
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
	for _, client := range r.Clients {
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
	r.Ticking = true
	game := &Game{
		Room: r,
	}
	r.Game = game
	game.init()

	for r.Ticking {
		game.tick()
		time.Sleep(time.Millisecond * 50)
	}
}

func (r *Room) stopGame() {
	r.Ticking = false
}

func (r *Room) onUserInputChange(client *Client, input int) bool {
	if r.Game == nil {
		return false
	}

	if !r.Game.isInRound() {
		return false
	}

	client.player.CurrentGuess = input

	return true
}

func (r *Room) onClientLeave(leavingClient *Client) {
	rc := r.Clients
	for i, c := range rc {
		if c == leavingClient {
			rc = append(rc[:i], rc[i+1:]...)
			break
		}
	}
	r.Clients = rc

	if len(r.Clients) == 1 {
		r.stopGame()
		r.sendMessageToAll(common.GameOverPacket{
			GenericMessage: common.GenericMessage{
				Type: common.GameOver,
			},
			Data: common.GameOverData{
				Winner: r.Clients[0].player,
			},
		})
	}
}
