package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"rlapi2mqtt/internal/port"
)

type StatusCallback func(msg string)

type ConnectionStateCallback func(mqttConnected, rlConnected bool)

type Companion struct {
	broker           port.BrokerClient
	game             port.GameClient
	onStatus         StatusCallback
	onConnState      ConnectionStateCallback
	topicPrefix      string
	logMessages      bool
	logFile          *os.File
	cancel           context.CancelFunc
	limitUpdateState bool
	lastUpdateState  time.Time
}

func NewCompanion(broker port.BrokerClient, game port.GameClient, logMessages bool) *Companion {
	return &Companion{
		broker:      broker,
		game:        game,
		onStatus:    func(string) {},
		onConnState: func(bool, bool) {},
		topicPrefix: "rlapi2mqtt",
		logMessages: logMessages,
	}
}

func (c *Companion) SetStatusCallback(cb StatusCallback) {
	c.onStatus = cb
}

func (c *Companion) SetConnectionStateCallback(cb ConnectionStateCallback) {
	c.onConnState = cb
}

func (c *Companion) StartLoop(brokerURL, username, password, rlAddress string) {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	if c.logMessages {
		f, err := os.OpenFile("messages.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			c.onStatus(fmt.Sprintf("Failed to open messages.log: %s", err.Error()))
		} else {
			c.logFile = f
		}
	}

	go c.connectLoop(ctx, brokerURL, username, password, rlAddress)
}

func (c *Companion) StopLoop() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.game.Disconnect()
	c.broker.Disconnect()
	if c.logFile != nil {
		c.logFile.Close()
		c.logFile = nil
	}
	c.onConnState(false, false)
	c.onStatus("Disconnected")
}

func (c *Companion) connectLoop(ctx context.Context, brokerURL, username, password, rlAddress string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	c.tryConnect(brokerURL, username, password, rlAddress)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.tryConnect(brokerURL, username, password, rlAddress)
		}
	}
}

func (c *Companion) tryConnect(brokerURL, username, password, rlAddress string) {
	if !c.broker.IsConnected() {
		c.onStatus("Connecting to MQTT broker " + brokerURL + "...")
		err := c.broker.Connect(brokerURL, username, password)
		if err != nil {
			c.onStatus("MQTT error: " + err.Error())
		} else {
			c.onStatus("MQTT connected to " + brokerURL)
		}
	}

	if !c.game.IsConnected() {
		c.onStatus("Connecting to Rocket League at " + rlAddress + "...")
		err := c.game.Connect(rlAddress, c.handleGameEvent)
		if err != nil {
			c.onStatus("RL error: " + err.Error())
		} else {
			c.onStatus("Rocket League connected to " + rlAddress)
		}
	}

	c.onConnState(c.broker.IsConnected(), c.game.IsConnected())
}

func (c *Companion) SetLimitUpdateState(enabled bool) {
	c.limitUpdateState = enabled
}

func (c *Companion) handleGameEvent(event port.GameEvent) {
	if c.limitUpdateState && strings.EqualFold(event.Event, "UpdateState") {
		now := time.Now()
		if now.Sub(c.lastUpdateState) < time.Second {
			return
		}
		c.lastUpdateState = now
	}

	topic := fmt.Sprintf("%s/%s", c.topicPrefix, strings.ToLower(event.Event))

	preview := string(event.Data)
	if len(preview) > 120 {
		preview = preview[:120] + "..."
	}
	c.onStatus(fmt.Sprintf("[%s] %s", event.Event, preview))

	if c.logFile != nil {
		c.logFile.Write(event.Data)
		c.logFile.Write([]byte("\n"))
	}

	err := c.broker.Publish(topic, event.Data)
	if err != nil {
		c.onStatus(fmt.Sprintf("Publish error [%s]: %s", event.Event, err.Error()))
		return
	}
}

func (c *Companion) IsBrokerConnected() bool {
	return c.broker.IsConnected()
}

func (c *Companion) IsGameConnected() bool {
	return c.game.IsConnected()
}
