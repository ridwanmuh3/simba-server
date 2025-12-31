ENV_FILE ?= .env

COMPOSE = docker compose --env-file $(ENV_FILE)

run-dev:
	go run ./cmd/app/main.go

run-seed:
	go run ./cmd/seed/seeder.go

compose-up:
	$(COMPOSE) up -d

compose-down:
	$(COMPOSE) down

compose-clean:
	$(COMPOSE) down -v

compose-logs:
	$(COMPOSE) logs -f

atlas-inspect:
	atlas schema inspect --env gorm -u "env://src"

atlas-apply:
	atlas schema apply --env gorm -u "env://dev"

atlas-migrate:
	atlas migrate diff --env gorm 

