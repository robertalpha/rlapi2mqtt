package port

type BrokerClient interface {
	Connect(brokerURL string) error
	Disconnect()
	IsConnected() bool
}
