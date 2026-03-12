.PHONY: run build test lint

run:
	go run cmd/app/main.go

build:
	go build -o bin/app cmd/app/main.go

test:
	go test -v ./...

lint:
	golangci-lint run
