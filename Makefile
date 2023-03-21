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
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin 
