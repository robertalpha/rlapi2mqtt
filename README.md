# rlapi2mqtt

A lightweight Windows desktop application that bridges the Rocket League Stats API WebSocket to an MQTT broker.

It connects to the game's local WebSocket, receives real-time events (goals, demolitions, stat updates, etc.) and publishes them to configurable MQTT topics under `rlapi2mqtt/`.

## Communication Flow

```
┌──────────────┐  WebSocket   ┌──────────────┐   MQTT      ┌──────────────┐
│ Rocket League│─────────────>│  rlapi2mqtt  │────────────>│  Mosquitto   │
│    Game      │ :49123       │  (this app)  │             │   Broker     │
└──────────────┘              └──────────────┘             └──────┬───────┘
                                                                 │
                                                                 │ subscribe
                                                                 v
                                                          ┌──────────────┐
                                                          │ rlannouncer  │
                                                          │  (consumer)  │
                                                          └──────────────┘
```

## Screenshot

![rlapi2mqtt](rlapi2mqtt_screenshot.png)

## Configuration

Create a `.env` file in the project root (or set environment variables):

```env
MQTT_URL=tcp://localhost:1883
MQTT_USERNAME=user
MQTT_PASSWORD=password
RL_ADDRESS=127.0.0.1:49123
```

All fields can also be filled in through the GUI at runtime.

## Build

Requires Go 1.26+. The application targets Windows only (uses [windigo](https://github.com/rodrigocfd/windigo) for the native GUI).

```sh
make build
```

This produces `rlapi2mqtt.exe`.
