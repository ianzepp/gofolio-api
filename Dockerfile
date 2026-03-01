# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /server /server

EXPOSE 3333

CMD ["/server"]
