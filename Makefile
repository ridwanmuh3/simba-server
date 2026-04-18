# ROOT := $(abspath ..)

run-dev:
	APP_MODE=dev go run ./cmd/app/main.go

run-seed:
	APP_MODE=dev go run ./cmd/seed/seeder.go

run-migrate:
	APP_MODE=dev go run ./cmd/migrate/main.go

run-build:
	CGO_ENABLED=0 go build -o ./bin/app     ./cmd/app/main.go
	CGO_ENABLED=0 go build -o ./bin/migrate ./cmd/migrate/main.go
	CGO_ENABLED=0 go build -o ./bin/seed    ./cmd/seed/seeder.go

test:
	go test ./...

tidy:
	go mod tidy

# Delegate stack orchestration to the root Makefile.
# %:
# 	$(MAKE) -C $(ROOT) $@
