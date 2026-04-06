.PHONY: swagger run build test lint

MAIN_PATH=cmd/app/main.go
DOCS_PATH=docs

run:
	go run cmd/app/main.go

build: swagger
	@echo "Building..."
	go build -o bin/app cmd/app/main.go

test:
	go test -v ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...
	@which goimports > /dev/null 2>&1 || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	$(shell go env GOPATH)/bin/goimports -w -local "auth-service" .

lint:
	@[ -f $(shell go env GOPATH)/bin/golangci-lint ] || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@echo "Running linter..."
	$(shell go env GOPATH)/bin/golangci-lint run -c .golangci.yml

swagger:
	@which swag > /dev/null 2>&1 || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	@echo "Generating swagger docs..."
	$(shell go env GOPATH)/bin/swag init -g $(MAIN_PATH) -o $(DOCS_PATH)
	@echo "Done!"

docker-build:
	@make swagger
	docker-compose up -d --build auth-service
