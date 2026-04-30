.PHONY: build clean

BINARY_NAME=rla-companion.exe
CMD_PATH=./cmd/rla-companion
LDFLAGS=-s -w -H=windowsgui

build:
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_PATH)

clean:
	rm -f $(BINARY_NAME)
