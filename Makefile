.PHONY: up down deps dev migrate test race build benchmark integration
up:
	docker compose up --build -d
down:
	docker compose down
deps:
	docker compose up -d postgres redis nats prometheus grafana
dev:
	node scripts/dev.mjs
migrate:
	go run ./apps/api migrate
test:
	go test ./...
	npm test
race:
	go test -race ./...
build:
	go build ./apps/api
	go build ./apps/simulator
	npm run build
benchmark:
	go test -run=^$$ -bench=. -benchmem ./apps/api/internal/optimization ./apps/api/internal/simulation ./apps/api/internal/events
integration:
	go test -tags=integration -v ./apps/api/internal/transport
