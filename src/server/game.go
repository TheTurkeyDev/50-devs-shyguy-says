package server

import (
	"math/rand"
	"shyguy-says/src/common"
	"time"
)

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

type Game struct {
	room *Room
}

func (g *Game) init() {
}

func (g *Game) tick() {
	if seededRand.Intn(100) == 42 {
		g.room.sendMessageToAll(common.ShyGuyDisplayPacket{
			GenericMessage: common.GenericMessage{
				Type: common.ShyGuyDsiplay,
			},
			Data: common.ShyGuyDisplayData{
				Input: seededRand.Intn(2),
			},
		})
	}
}
