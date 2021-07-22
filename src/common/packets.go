package common

const (
	JoinRoomRequest = iota
	JoinRoomResponse
	CreateRoomRequest
	CreateRoomResponse
	UserJoin
	UserLeave
	UserInput
	StartGame
	StartGameResponse
	GameOver
	ShyGuyDsiplay
)

type RoomIdent struct {
	Name     string
	Password string
}

type GenericMessage struct {
	Type int
}

type JoinRoomRequestPacket struct {
	GenericMessage
	Data JoinRoomRequestData
}

type JoinRoomRequestData struct {
	Room        RoomIdent
	DisplayName string
}

type JoinRoomResponsePacket struct {
	GenericMessage
	Data JoinRoomResponseData
}

type JoinRoomResponseData struct {
	Room    RoomIdent
	Valid   bool
	Message string
	MyId    string
	Players []Player
}

type UserJoinRoomData struct {
	Room   RoomIdent
	Player Player
}

type UserJoinRoomPacket struct {
	GenericMessage
	Data UserJoinRoomData
}

type UserLeaveRoomData struct {
	PlayerId string
	Reason   string
}

type UserLeaveRoomPacket struct {
	GenericMessage
	Data UserLeaveRoomData
}

type UserInputData struct {
	PlayerId string
	Input    int
}

type UserInputPacket struct {
	GenericMessage
	Data UserInputData
}

type StartGameData struct {
	Room RoomIdent
}

type StartGamePacket struct {
	GenericMessage
	Data StartGameData
}

type StartGameResponseData struct {
	Valid   bool
	Message string
}

type StartGameResponsePacket struct {
	GenericMessage
	Data StartGameResponseData
}

type ShyGuyDisplayData struct {
	Input int
}

type ShyGuyDisplayPacket struct {
	GenericMessage
	Data ShyGuyDisplayData
}

type GameOverData struct {
	Winner Player
}

type GameOverPacket struct {
	GenericMessage
	Data GameOverData
}
