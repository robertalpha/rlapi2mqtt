package main

import (
	"rla-companion/internal/adapter/broker"
	"rla-companion/internal/adapter/gui"
)

func main() {
	mqttClient := broker.NewMQTTClient()
	gui.Run(mqttClient)
}
