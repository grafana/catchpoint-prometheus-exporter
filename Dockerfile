FROM golang:1.23-alpine@sha256:383395b794dffa5b53012a212365d40c8e37109a626ca30d6151c8348d380b5f AS builder

WORKDIR /app
COPY . .

RUN go build -o exporter ./cmd/catchpoint-exporter/main.go

FROM alpine:latest@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412

COPY --from=builder /app/exporter /exporter

ENTRYPOINT ["/exporter"]
