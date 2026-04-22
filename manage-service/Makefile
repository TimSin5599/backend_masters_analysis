# Database URL for migrations
MG_URL="postgres://user:password@localhost:5433/fqw_db?sslmode=disable"
MAIN_PATH=cmd/app/main.go
DOCS_PATH=docs

.PHONY: run build test lint fmt vet swagger migrate-up

## run: Run the service locally
run:
	go run cmd/app/main.go

## build: Build the binary
build:
	go build -o bin/app cmd/app/main.go

## test: Run all tests
test:
	go test -v -race ./...

## fmt: Format code with gofmt
fmt:
	gofmt -w .

## vet: Run go vet
vet:
	go vet ./...

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## swagger: Generate Swagger docs (requires swag: go install github.com/swaggo/swag/cmd/swag@latest)
swagger:
	swag init -g cmd/app/main.go --parseDependency --parseInternal

## migrate-up: Apply all up migrations
migrate-up:
	docker run --rm -v $(shell pwd)/migrations:/migrations --network fqw_default migrate/migrate \
		-path=/migrations/ -database "$(MG_URL)" up

swagger:
	@which swag > /dev/null 2>&1 || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	@echo "Generating swagger docs..."
	$(shell go env GOPATH)/bin/swag init -g $(MAIN_PATH) -o $(DOCS_PATH)
	@echo "Done!"

docker-build:
	@make swagger
	docker-compose up -d --build manage-service
