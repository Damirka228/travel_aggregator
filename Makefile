DB_URL=postgres://travel:travel@localhost:5432/travel_aggregator?sslmode=disable

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

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

db-logs:
	docker-compose logs -f postgres

migrate-up:
	goose -dir migrations postgres "$(DB_URL)" up
migrate-down:
	goose -dir migrations postgres "$(DB_URL)" down
