FROM mcr.microsoft.com/devcontainers/base:ubuntu-22.04

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG GO_VERSION=1.23.4
ARG FLUTTER_VERSION=3.27.0
ARG TERRAFORM_VERSION=1.9.8
ARG BUF_VERSION=1.47.2
ARG NODE_MAJOR=22

RUN apt-get update \
    && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
        apt-transport-https \
        ca-certificates \
        curl \
        git \
        gnupg \
        unzip \
        xz-utils \
    && rm -rf /var/lib/apt/lists/*

# Go
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
    | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

# Node.js
RUN curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

# Dart + Flutter
RUN git clone --depth 1 --branch "${FLUTTER_VERSION}" \
    https://github.com/flutter/flutter.git /opt/flutter
ENV PATH="/opt/flutter/bin:${PATH}"
RUN flutter config --no-analytics \
    && flutter precache --web

# Terraform
RUN curl -fsSL "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip" \
    -o /tmp/terraform.zip \
    && unzip /tmp/terraform.zip -d /usr/local/bin \
    && rm /tmp/terraform.zip

# Buf
RUN curl -fsSL "https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/buf-Linux-x86_64" \
    -o /usr/local/bin/buf \
    && chmod +x /usr/local/bin/buf

# kubectl + helm + k9s + gcloud CLI
RUN curl -fsSL https://dl.k8s.io/release/stable.txt | xargs -I{} \
    curl -fsSL "https://dl.k8s.io/release/{}/bin/linux/amd64/kubectl" \
    -o /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl

RUN curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

RUN curl -fsSL https://github.com/derailed/k9s/releases/latest/download/k9s_Linux_amd64.tar.gz \
    | tar -C /usr/local/bin -xz k9s

RUN curl -fsSL https://sdk.cloud.google.com | bash -s -- --disable-prompts \
    --install-dir=/opt/google-cloud-sdk
ENV PATH="/opt/google-cloud-sdk/bin:${PATH}"

WORKDIR /workspace

