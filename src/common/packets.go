package common

const (
	JoinRoomRequest = iota
	JoinRoomResponse
	CreateRoomRequest
	CreateRoomResponse
	UserJoin
	UserLeave
	UserInput
)

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

type RoomIdent struct {
	Name     string
	Password string
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
