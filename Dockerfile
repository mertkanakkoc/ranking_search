# --- build stage ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api ./cmd/api

# --- final stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/api /bin/api

EXPOSE 8080

ENTRYPOINT ["/bin/api"]
