package port

type GameEvent struct {
	Event string
	Data  []byte
}

type GameEventCallback func(event GameEvent)

type GameClient interface {
	Connect(address string, onEvent GameEventCallback) error
	Disconnect()
	IsConnected() bool
}
