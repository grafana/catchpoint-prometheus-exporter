FROM golang:1.20-alpine@sha256:e47f121850f4e276b2b210c56df3fda9191278dd84a3a442bfe0b09934462a8f AS builder

WORKDIR /app
COPY . .

RUN go build -o exporter ./cmd/catchpoint-exporter/main.go

FROM alpine:latest@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412

COPY --from=builder /app/exporter /exporter

ENTRYPOINT ["/exporter"]
