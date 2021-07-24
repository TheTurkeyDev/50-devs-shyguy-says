package server

import (
	"fmt"
	"math/rand"
	"shyguy-says/src/common"
	"time"
)

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

type Game struct {
	Room        *Room
	Round       int
	RoundStatus int
	Delay       int
	ShyGuyPick  int
}

func (g *Game) init() {
	g.Round = 0
	for _, c := range g.Room.Clients {
		c.player.Elimated = false
	}
	g.nextRound(false)
}

func (g *Game) tick() {
	if g.Delay == 0 {
		switch g.RoundStatus {
		// Pre-Round
		case 0:
			g.onRoundStart()
		// In round
		case 1:
			g.onRoundEnd()
		// Post-Round
		case 2:
			g.nextRound(true)
		}
	}

	switch g.RoundStatus {
	// Pre-Round
	case 0:
		if g.Delay <= 100 && g.Delay%20 == 0 {
			val := fmt.Sprintf("%d", g.Delay/20)
			if val == "0" {
				val = ""
			}
			g.Room.sendMessageToAll(common.TitleMessagePacket{
				GenericMessage: common.GenericMessage{
					Type: common.TitleMessage,
				},
				Data: common.TitleMessageData{
					Value:       val,
					Color:       "#ff0000",
					Location:    0,
					DisplayTime: 20,
				},
			})
		}
	// In round
	case 1:
		if (seededRand.Intn(100) < 5 && g.Delay > 40) || (g.ShyGuyPick == -1 && g.Delay == 40) {
			g.ShyGuyPick = seededRand.Intn(2)
			g.Room.sendMessageToAll(common.ShyGuyDisplayPacket{
				GenericMessage: common.GenericMessage{
					Type: common.ShyGuyDsiplay,
				},
				Data: common.ShyGuyDisplayData{
					Input: g.ShyGuyPick,
				},
			})
		}
	// Post-Round
	case 2:

	}

	g.Delay -= 1
}

func (g *Game) onRoundStart() {
	g.setRoundAndStatus(g.Round, 1)
	g.Delay = seededRand.Intn(100) + 60
	g.ShyGuyPick = -1
}

func (g *Game) onRoundEnd() {
	g.setRoundAndStatus(g.Round, 2)
	g.Delay = 100
}

func (g *Game) nextRound(endCheck bool) {
	alive := 0
	for _, c := range g.Room.Clients {
		if endCheck && c.player.CurrentGuess != g.ShyGuyPick {
			c.player.Elimated = true
			// TODO eliminate packet
		} else {
			alive += 1
		}
	}

	if alive == 1 {
		g.Room.stopGame()
		for _, c := range g.Room.Clients {
			if !c.player.Elimated {
				g.sendGameOverPackets(c.player)
				break
			}
		}
	} else if alive == 0 {
		g.Room.stopGame()
		g.sendGameOverPackets(common.Player{
			Id:           "",
			CurrentGuess: -1,
			Elimated:     false,
			PlayerNum:    -1,
			DisplayName:  "No one",
		})
	} else {
		g.setRoundAndStatus(g.Round+1, 0)
		g.Delay = 100
		g.ShyGuyPick = -1
	}

	for _, c := range g.Room.Clients {
		c.player.CurrentGuess = -1
		g.Room.sendMessageToAll(common.UserInputPacket{
			GenericMessage: common.GenericMessage{
				Type: common.UserInput,
			},
			Data: common.UserInputData{
				PlayerId: c.player.Id,
				Input:    -1,
			},
		})
	}

	g.Room.sendMessageToAll(common.ShyGuyDisplayPacket{
		GenericMessage: common.GenericMessage{
			Type: common.ShyGuyDsiplay,
		},
		Data: common.ShyGuyDisplayData{
			Input: -1,
		},
	})
}

func (g *Game) isInRound() bool {
	return g.RoundStatus == 1
}

func (g *Game) setRoundAndStatus(round int, status int) {
	g.Round = round
	g.RoundStatus = status
	g.Room.sendMessageToAll(common.RoundUpdatePacket{
		GenericMessage: common.GenericMessage{
			Type: common.RoundUpdate,
		},
		Data: common.RoundUpdateData{
			Round:       g.Round,
			RoundStatus: g.RoundStatus,
		},
	})
}

func (g *Game) sendGameOverPackets(winner common.Player) {
	g.Room.sendMessageToAll(common.GameOverPacket{
		GenericMessage: common.GenericMessage{
			Type: common.GameOver,
		},
		Data: common.GameOverData{
			Winner: winner,
		},
	})
	g.Room.sendMessageToAll(common.TitleMessagePacket{
		GenericMessage: common.GenericMessage{
			Type: common.TitleMessage,
		},
		Data: common.TitleMessageData{
			Value:       "Game Over!",
			Color:       "#ff0000",
			Location:    0,
			DisplayTime: 100,
		},
	})
	g.Room.sendMessageToAll(common.TitleMessagePacket{
		GenericMessage: common.GenericMessage{
			Type: common.TitleMessage,
		},
		Data: common.TitleMessageData{
			Value:       winner.DisplayName + " won!",
			Color:       "#ff0000",
			Location:    1,
			DisplayTime: 100,
		},
	})
}
