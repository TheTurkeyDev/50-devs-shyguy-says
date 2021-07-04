package main

type GameState struct {
	InRoom  bool
	Players []Player
	MyId    int
}

func (s *GameState) SetInRoom(inRoom bool) {
	s.InRoom = inRoom
}
