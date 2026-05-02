.PHONY: build clean

BINARY_NAME=rlapi2mqtt.exe
CMD_PATH=./cmd/rlapi2mqtt
LDFLAGS=-s -w -H=windowsgui

build:
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_PATH)

clean:
	rm -f $(BINARY_NAME)
