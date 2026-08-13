.PHONY: up down test run lint

up:
	docker compose up -d

down:
	docker compose down

test:
	go test ./... -v -race -count=1

run:
	go run ./cmd/server/main.go

lint:
	golangci-lint run ./...