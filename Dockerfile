FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /collector ./cmd/collector

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /collector .
# Copy migrations for potential runtime use or inspection
COPY --from=builder /app/migrations ./migrations

EXPOSE 3100

CMD ["./collector"]
