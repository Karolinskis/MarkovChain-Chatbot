FROM golang:1.26-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bot ./cmd/bot
RUN go build -o migrate ./cmd/migrate

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/bot /bot
COPY --from=builder /app/migrate /migrate
COPY --from=builder /app/internal/database/migrations /internal/database/migrations

ENTRYPOINT ["/bot"]
