.DEFAULT_GOAL := all

NAME := $(shell basename $(CURDIR))

all: build test format lint

clean:
	@echo "Cleaning ${NAME}..."
	@go clean -i ./...
	@rm -rf bin fyne-cross

build: clean
	@echo "Building ${NAME}..."
	@GOOS=darwin GOARCH=amd64 go build -o ./bin/${NAME}_cli_darwin-amd64 ./cmd/cli
	@GOOS=windows GOARCH=amd64 go build -o ./bin/${NAME}_cli_windows-amd64.exe ./cmd/cli
	@GOOS=linux GOARCH=amd64 go build -o ./bin/${NAME}_cli_linux-amd64 ./cmd/cli
	@fyne-cross windows -arch=amd64 --app-id playlist.creator ./cmd/gui
	@mv ./fyne-cross/bin/windows-amd64/playlist-creator.exe ./bin/${NAME}_gui_windows-amd64.exe

test: build
	@echo "Testing ${NAME}..."
	@gotestsum ./... -cover -race -shuffle=on

format:
	@echo "Formatting ${NAME}..."
	@go mod tidy
	@gofumpt -l -w .

lint:
	@echo "Linting ${NAME}..."
	@go vet ./...
	@govulncheck ./...
	@golangci-lint run

deps:
	@echo "Installing ${NAME} dependencies..."
	@go install gotest.tools/gotestsum@latest
	@go install mvdan.cc/gofumpt@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install fyne.io/fyne/v2/cmd/fyne@latest
	@go install github.com/fyne-io/fyne-cross@latest
