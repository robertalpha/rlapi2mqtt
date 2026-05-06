package main

import (
	"os"

	"rlapi2mqtt/internal/adapter/broker"
	"rlapi2mqtt/internal/adapter/game"
	"rlapi2mqtt/internal/adapter/gui"
	"rlapi2mqtt/internal/service"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("rlapi2mqtt.ini")

	cfg := gui.Config{
		MQTTUrl:      os.Getenv("MQTT_URL"),
		MQTTUsername: os.Getenv("MQTT_USERNAME"),
		MQTTPassword: os.Getenv("MQTT_PASSWORD"),
		RLAddress:    os.Getenv("RL_ADDRESS"),
		AutoConnect:  os.Getenv("AUTO_CONNECT") == "true",
	}

	mqttClient := broker.NewMQTTClient()
	gameClient := game.NewRLClient()
	companion := service.NewCompanion(mqttClient, gameClient)
	gui.Run(companion, cfg)
}
