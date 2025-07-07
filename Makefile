.PHONY: build run test clean deps docker-build

BINARY_NAME=loopgate
CMD_PATH=./cmd/server

build:
	go build -o $(BINARY_NAME) $(CMD_PATH)

run:
	go run $(CMD_PATH)

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out

deps:
	go mod download
	go mod tidy

docker-build:
	docker build -t $(BINARY_NAME):latest .

docker-compose:
	docker-compose up -d

.DEFAULT_GOAL := build