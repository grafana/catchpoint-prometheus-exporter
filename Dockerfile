FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o exporter ./cmd/catchpoint-exporter/main.go

FROM alpine:latest

COPY --from=builder /app/exporter /exporter

ENTRYPOINT ["/exporter"]
