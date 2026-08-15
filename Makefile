.PHONY: up down test run build lint

up:
	docker compose up -d

down:
	docker compose down

test:
	go test ./... -v -race -count=1

run:
	go run ./cmd/server/main.go

build:
	go build -o bin/dlm ./cmd/cli/main.go
	go build -o bin/server ./cmd/server/main.go

lint:
	golangci-lint run ./....PHONY: up down test run build lint

up:
	docker compose up -d

down:
	docker compose down

test:
	go test ./... -v -race -count=1

run:
	go run ./cmd/server/main.go

build:
	go build -o bin/dlm ./cmd/cli/main.go
	go build -o bin/server ./cmd/server/main.go

lint:
	golangci-lint run ./...