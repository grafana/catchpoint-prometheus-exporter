FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o exporter ./cmd/catchpoint-exporter/main.go

FROM alpine:latest@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412

COPY --from=builder /app/exporter /exporter

ENTRYPOINT ["/exporter"]

