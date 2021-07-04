package common

const (
	JoinRoom = iota
	JoinRoomResult
	CreateRoom
	CreateRoomResult
)

type GenericMessage struct {
	Type int
}

type JoinRoomMessage struct {
	GenericMessage
	Data RoomIdent
}

type RoomIdent struct {
	Name     string
	Password string
}

type JoinRoomResultMessage struct {
	GenericMessage
	Data JoinResult
}

type JoinResult struct {
	Room    RoomIdent
	Valid   bool
	Message string
}
