package broker

import (
	"fmt"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
	client mqtt.Client
}

func NewMQTTClient() *MQTTClient {
	return &MQTTClient{}
}

func (m *MQTTClient) Connect(brokerURL, username, password string) error {
	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(fmt.Sprintf("rlapi2mqtt-%d", rand.Intn(90000)+10000)).
		SetConnectTimeout(5 * time.Second)

	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}

	m.client = mqtt.NewClient(opts)
	token := m.client.Connect()
	token.WaitTimeout(5 * time.Second)

	if err := token.Error(); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	if !m.client.IsConnected() {
		return fmt.Errorf("connection timed out — broker did not respond within 5 seconds")
	}

	return nil
}

func (m *MQTTClient) Disconnect() {
	if m.client != nil && m.client.IsConnected() {
		m.client.Disconnect(250)
	}
}

func (m *MQTTClient) IsConnected() bool {
	return m.client != nil && m.client.IsConnected()
}

func (m *MQTTClient) Publish(topic string, payload []byte) error {
	if m.client == nil || !m.client.IsConnected() {
		return fmt.Errorf("not connected to broker")
	}
	token := m.client.Publish(topic, 0, false, payload)
	token.WaitTimeout(2 * time.Second)
	if err := token.Error(); err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}
	return nil
}
