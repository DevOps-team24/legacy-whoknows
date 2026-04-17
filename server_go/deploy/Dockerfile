# Build stage
FROM golang:1.25.9-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -o whoknows-server ./cmd/server

# Runtime stage
FROM alpine:3.21

# hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates sqlite-libs

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
