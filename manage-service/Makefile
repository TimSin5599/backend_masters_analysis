# Database URL for migrations
MG_URL="postgres://user:password@localhost:5433/fqw_db?sslmode=disable"

.PHONY: run build test lint migrate-up

run:
	go run cmd/app/main.go

build:
	go build -o bin/app cmd/app/main.go

test:
	go test -v ./...

lint:
	golangci-lint run

migrate-up:
	docker run --rm -v $(shell pwd)/migrations:/migrations --network fqw_default migrate/migrate -path=/migrations/ -database "$(MG_URL)" up
