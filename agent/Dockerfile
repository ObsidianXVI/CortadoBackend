FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o cortado-agent ./cmd/agent

FROM ubuntu:22.04

ARG INCLUDE_DART_SDK=false
ARG DART_SDK_VERSION=3.11.5

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        file \
        git \
        python3 \
        restic \
        unzip \
        vim \
        wget \
        zsh \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1000 workspace

RUN if [ "$INCLUDE_DART_SDK" = "true" ]; then \
      wget -q "https://storage.googleapis.com/dart-archive/channels/stable/release/${DART_SDK_VERSION}/sdk/dartsdk-linux-x64-release.zip" -O /tmp/dart-sdk.zip \
      && unzip -q /tmp/dart-sdk.zip -d /usr/local \
      && rm /tmp/dart-sdk.zip \
      && ln -sf /usr/local/dart-sdk/bin/dart /usr/local/bin/dart \
      && ln -sf /usr/local/dart-sdk/bin/dartaotruntime /usr/local/bin/dartaotruntime; \
    fi

COPY --from=builder /build/cortado-agent /usr/local/bin/cortado-agent

EXPOSE 9090

ENV CORTADO_ENV=production
ENV DART_SDK_PATH=/usr/local/dart-sdk
ENV PATH=/usr/local/dart-sdk/bin:${PATH}

ENTRYPOINT ["/usr/local/bin/cortado-agent"]
