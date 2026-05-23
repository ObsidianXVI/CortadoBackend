FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o cortado-agent ./cmd/agent

FROM ubuntu:22.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        file \
        git \
        python3 \
        vim \
        zsh \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1000 workspace

COPY --from=builder /build/cortado-agent /usr/local/bin/cortado-agent

EXPOSE 9090

ENV CORTADO_ENV=production

ENTRYPOINT ["/usr/local/bin/cortado-agent"]
