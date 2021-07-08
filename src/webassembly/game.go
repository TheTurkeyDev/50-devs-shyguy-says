package main

import "shyguy-says/src/common"

type GameState struct {
	InRoom  bool
	Players map[string]*common.Player
	MyId    string
	Room    common.RoomIdent
}

func (s *GameState) SetInRoom(inRoom bool) {
	s.InRoom = inRoom
}
