APP=tsmap-extract
SRC=$(APP).go
BIN_DIR=bin
GO=go

.PHONY: all tidy build clean linux windows mac

all: tidy build

tidy:
	$(GO) mod tidy

build: linux windows mac

linux: $(SRC)
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BIN_DIR)/$(APP)-linux-amd64 $(SRC)
	GOOS=linux GOARCH=arm64 $(GO) build -o $(BIN_DIR)/$(APP)-linux-arm64 $(SRC)

windows: $(SRC)
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BIN_DIR)/$(APP)-windows-amd64.exe $(SRC)

mac: $(SRC)
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BIN_DIR)/$(APP)-darwin-amd64 $(SRC)
	GOOS=darwin GOARCH=arm64 $(GO) build -o $(BIN_DIR)/$(APP)-darwin-arm64 $(SRC)

clean:
	rm -rf $(BIN_DIR)

