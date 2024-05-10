FROM alpine

COPY exporter /exporter

ENTRYPOINT ["/exporter"]
