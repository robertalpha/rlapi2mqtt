package main

import (
	"os"

	"rlapi2mqtt/internal/adapter/broker"
	"rlapi2mqtt/internal/adapter/game"
	"rlapi2mqtt/internal/adapter/gui"
	"rlapi2mqtt/internal/service"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	cfg := gui.Config{
		MQTTUrl:      os.Getenv("MQTT_URL"),
		MQTTUsername: os.Getenv("MQTT_USERNAME"),
		MQTTPassword: os.Getenv("MQTT_PASSWORD"),
		RLAddress:    os.Getenv("RL_ADDRESS"),
	}

	mqttClient := broker.NewMQTTClient()
	gameClient := game.NewRLClient()
	logMessages := os.Getenv("LOG_MESSAGES") == "true"
	companion := service.NewCompanion(mqttClient, gameClient, logMessages)
	gui.Run(companion, cfg)
}
