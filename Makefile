.PHONY: up down dev tailwind-watch build dev-up dev-down

up:
	docker compose up -d

down:
	docker compose down

dev-up:
	docker compose -f docker-compose.dev.yml up --build -d

dev-down:
	docker compose -f docker-compose.dev.yml down

dev:
	$(shell go env GOPATH)/bin/air

tailwind-watch:
	npx -p tailwindcss@3 tailwindcss -i internal/ui/input.css -o ui/style.css --watch
