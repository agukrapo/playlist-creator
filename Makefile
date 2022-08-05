.DEFAULT_GOAL := all

NAME := $(shell basename $(CURDIR))

all: build test format lint

clean:
	@echo "Cleaning ${NAME}..."
	@go clean -i ./...
	@rm -rf bin

build: clean
	@echo "Building ${NAME}..."
	@GOOS=darwin GOARCH=amd64 go build -o ./bin/${NAME}_darwin-amd64 ./cmd
	@GOOS=windows GOARCH=amd64 go build -o ./bin/${NAME}_windows-amd64.exe ./cmd
	@GOOS=linux GOARCH=amd64 go build -o ./bin/${NAME}_linux-amd64 ./cmd

test: build
	@echo "Testing ${NAME}..."
	@go test ./... -cover -race -shuffle=on

format:
	@echo "Formatting ${NAME}..."
	@go mod tidy
	@gofumpt -l -w . #go install mvdan.cc/gofumpt@latest

lint:
	@echo "Linting ${NAME}..."
	@go vet ./...
	@golangci-lint run #https://golangci-lint.run/usage/install/
