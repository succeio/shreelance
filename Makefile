.PHONY: up down dev tailwind-watch build

up:
	docker compose up -d

down:
	docker-compose down

dev:
	$(shell go env GOPATH)/bin/air

tailwind-watch:
	tailwindcss -i internal/ui/input.css -o ui/style.css --watch

build:
	go build -o bin/server cmd/server/main.go
