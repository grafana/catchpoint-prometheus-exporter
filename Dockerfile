FROM golang:1.26-alpine@sha256:27f2d3b83cfc5803c8078f8e730af0f7c431bf067afaec8f60a36fb763187cba AS builder

WORKDIR /app
COPY . .

RUN go build -o exporter ./cmd/catchpoint-exporter/main.go

FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

COPY --from=builder /app/exporter /exporter

ENTRYPOINT ["/exporter"]
