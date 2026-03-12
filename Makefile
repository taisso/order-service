APP_NAME := order-service
BIN := bin/$(APP_NAME)

# Caminhos mais críticos para cobertura
COVER_PKGS := ./internal/domain/...,./internal/application/...,./internal/adapters/...

.PHONY: run dev build test coverage coverage-html swagger lint docker-up docker-down migrate

run:
	go run ./cmd/server

dev:
	go tool air

build:
	go build -o $(BIN) ./cmd/server

test:
	go test ./... -coverpkg=$(COVER_PKGS) -coverprofile=coverage.out

coverage: test
	go tool cover -func=coverage.out

coverage-html:
	go tool cover -html=coverage.out -o coverage.html

swagger:
	go tool swag init -g cmd/server/main.go -o docs

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down


