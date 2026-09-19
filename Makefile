run:
	go run ./cmd/app

test:
	go test ./...

test-race:
	go test -race ./...

build:
	go build -o bin/app ./cmd/app

tidy:
	go mod tidy

.PHONY: run test test-race build tidy