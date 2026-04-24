FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init -g cmd/app/main.go

RUN go build -o main cmd/app/main.go

FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations

CMD ["./main"]
