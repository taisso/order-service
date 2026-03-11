FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o order-service ./cmd/server

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/order-service /app/order-service
COPY --from=builder /app/.env /app/.env

EXPOSE 8080

ENTRYPOINT ["/app/order-service"]

