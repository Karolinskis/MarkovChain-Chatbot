FROM golang:1.24-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bot .
RUN go build -o migrate ./cmd/migrate

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/bot /bot
COPY --from=builder /app/migrate /migrate
COPY --from=builder /app/database/migrations /database/migrations

ENTRYPOINT ["/bot"]
