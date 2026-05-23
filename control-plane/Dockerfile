FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY control-plane/go.mod control-plane/go.sum ./control-plane/
COPY agent/go.mod agent/go.sum ./agent/

WORKDIR /build/control-plane
RUN go mod download

WORKDIR /build
COPY agent ./agent
COPY control-plane ./control-plane

WORKDIR /build/control-plane
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o cortado-control-plane ./cmd/server

FROM ubuntu:22.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        tzdata \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/control-plane/cortado-control-plane /usr/local/bin/cortado-control-plane

EXPOSE 8080

ENV CORTADO_ENV=production
ENV PORT=8080

ENTRYPOINT ["/usr/local/bin/cortado-control-plane"]
