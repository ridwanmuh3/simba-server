ENV_FILE ?= .env
COMPOSE = docker compose --env-file $(ENV_FILE)

run-dev:
	APP_MODE=dev go run ./cmd/app/main.go

run-seed:
	APP_MODE=dev go run ./cmd/seed/seeder.go

run-build:
	APP_MODE=dev CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/app ./cmd/app/main.go && chmod +x ./bin/app && ./bin/app

compose-up:
	$(COMPOSE) up -d

compose-down:
	$(COMPOSE) down

compose-clean:
	$(COMPOSE) down -v

compose-logs:
	$(COMPOSE) logs -f

docker-build:
	docker build -t simba-api-server:latest .

docker-run:
	docker run -p 3000:3000 --env-file .env simba-api-server:latest

atlas-inspect:
	atlas schema inspect --env gorm -u "env://src"

atlas-apply:
	atlas schema apply --env gorm -u "env://dev"

atlas-clean:
	atlas schema apply --env gorm --to file://empty.hcl -u "env://dev"

atlas-migrate:
	atlas migrate diff --env gorm 

