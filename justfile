set dotenv-load := true

default:
    @just --list

# Run backend server locally
run-dev:
    APP_MODE=dev go run ./cmd/app/main.go

# Run seeder locally
run-seed:
    APP_MODE=dev go run ./cmd/seed/seeder.go

# Run migrations locally
run-migrate:
    APP_MODE=dev go run ./cmd/migrate/main.go

# Build app, migrate, and seed binaries
run-build:
    CGO_ENABLED=0 go build -o ./bin/app ./cmd/app/main.go
    CGO_ENABLED=0 go build -o ./bin/migrate ./cmd/migrate/main.go
    CGO_ENABLED=0 go build -o ./bin/seed ./cmd/seed/seeder.go

# Run backend tests
test:
    go test ./...

# Run go mod tidy
tidy:
    go mod tidy
