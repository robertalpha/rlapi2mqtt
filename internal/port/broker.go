package port

type BrokerClient interface {
	Connect(brokerURL, username, password string) error
	Disconnect()
	IsConnected() bool
	Publish(topic string, payload []byte) error
}
