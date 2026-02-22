# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -o whoknows-server ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite

WORKDIR /app

COPY --from=builder /build/whoknows-server .
COPY --from=builder /build/templates ./templates
COPY --from=builder /build/static ./static
COPY --from=builder /build/migrations ./migrations

RUN mkdir -p /app/data && chmod 755 /app/data

EXPOSE 8080

ENV WHOKNOWS_DB_PATH=/app/data/whoknows.db
ENV WHOKNOWS_PORT=8080
ENV WHOKNOWS_ADDR=0.0.0.0

CMD ["./whoknows-server"]
