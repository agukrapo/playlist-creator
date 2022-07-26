.DEFAULT_GOAL := lint

NAME := $(shell basename $(CURDIR))
VERSION := $(shell git describe --abbrev=0 --tags)

clean:
	@echo "Cleaning ${NAME}-${VERSION}..."
	@go clean -i ./...
	@rm -rf bin

build: clean
	@echo "Building ${NAME}-${VERSION}..."
	@GOOS=darwin GOARCH=amd64 go build -o ./bin/${NAME}-${VERSION}_darwin-amd64 ./cmd
	@GOOS=windows GOARCH=amd64 go build -o ./bin/${NAME}-${VERSION}_windows-amd64.exe ./cmd
	@GOOS=linux GOARCH=amd64 go build -o ./bin/${NAME}-${VERSION}_linux-amd64 ./cmd

test: build
	@echo "Testing ${NAME}-${VERSION}..."
	@go test ./... -cover -race -shuffle=on

format: test
	@echo "Formatting ${NAME}-${VERSION}..."
	@go mod tidy
	@gofumpt -l -w . #go install mvdan.cc/gofumpt@latest

lint: format
	@echo "Linting ${NAME}-${VERSION}..."
	@go vet ./...
	@golangci-lint run #https://golangci-lint.run/usage/install/
