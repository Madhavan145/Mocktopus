SHELL := /bin/sh

.PHONY: dev up down rebuild logs ps test lint migrate seed proto

dev: up

up:
	docker compose up -d --build

down:
	docker compose down

rebuild:
	docker compose up -d --build --force-recreate

logs:
	docker compose logs -f --tail=200

ps:
	docker compose ps

test:
	cd services/auth-service && go test ./...
	cd services/gateway-service && go test ./...
	cd services/topology-service && go test ./...
	cd services/simulation-service && go test ./...
	cd services/metrics-service && go test ./...
	cd services/cost-service && go test ./...
	cd apps/frontend && npm test -- --watch=false || true

lint:
	cd apps/frontend && npm run lint

migrate:
	echo "Migrations are currently managed by infra/postgres/init.sql for local dev"

seed:
	echo "Seeding currently handled by default demo topology in topology-service"

proto:
	sh scripts/proto-gen.sh
