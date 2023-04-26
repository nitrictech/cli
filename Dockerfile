FROM golang:1.19.4-buster as build

WORKDIR /app/

RUN apt-get update -y && \
  apt-get install --no-install-recommends -y curl && \
  curl -fsSL https://deb.nodesource.com/setup_16.x | bash - &&\
  curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && \
  echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
  apt-get update -y && \
  apt-get install -y nodejs yarn 

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build

FROM python:3.9-slim

LABEL "repository"="https://github.com/nitrictech/cli"
LABEL "homepage"="https://nitric.io"
LABEL org.opencontainers.image.description="The Nitric CLI, in a Docker container."

ENV GOLANG_VERSION 1.17.3
ENV GOLANG_SHA256 550f9845451c0c94be679faf116291e7807a8d78b43149f9506c1b15eb89008c
ENV DOCKER_PASS_CH v0.6.4

ARG DOCKER_VERSION=5:20.10.22~3-0~debian-bullseye

# Install deps all in one step
RUN apt-get update -y && \
  apt-get install --no-install-recommends -y \
  apt-transport-https \
  build-essential \
  ca-certificates \
  net-tools \
  curl \
  git \
  gnupg \
  software-properties-common \
  wget \
  pass \
  unzip && \
  # Get all of the signatures we need all at once.
  curl -fsSL https://download.docker.com/linux/debian/gpg          | apt-key add - && \
  # IAM Authenticator for EKS
  curl -fsSLo /usr/bin/aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-07-26/bin/linux/amd64/aws-iam-authenticator && \
  chmod +x /usr/bin/aws-iam-authenticator && \
  # AWS v2 cli
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
  unzip awscliv2.zip && \
  ./aws/install && \
  rm -rf aws && \
  # Add additional apt repos all at once
  echo "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"      | tee /etc/apt/sources.list.d/docker.list && \
  # Install second wave of dependencies
  apt-get update -y && \
  apt-get install -y docker-ce=$DOCKER_VERSION docker-ce-cli=$DOCKER_VERSION && \
  # Clean up the lists work
  rm -rf /var/lib/apt/lists/*

# Passing --build-arg PULUMI_VERSION=vX.Y.Z will use that version
# of the SDK. Otherwise, we use whatever get.pulumi.com thinks is
# the latest
ARG PULUMI_VERSION=3.49.0

# Install the Pulumi SDK, including the CLI and language runtimes.
RUN curl -fsSL https://get.pulumi.com/ | bash -s -- --version $PULUMI_VERSION && \
  mv ~/.pulumi/bin/* /usr/bin

RUN curl -fsSLo /tmp/dch.tgz https://github.com/docker/docker-credential-helpers/releases/download/${DOCKER_PASS_CH}/docker-credential-pass-${DOCKER_PASS_CH}-amd64.tar.gz; \
  tar -xf /tmp/dch.tgz; \
  chmod +x docker-credential-pass; \
  mv -f docker-credential-pass /usr/bin/; \
  rm -rf /tmp/dch.tgz

ENV HOST_DOCKER_INTERNAL_IFACE eth0
ENV PULUMI_SKIP_UPDATE_CHECK "true"

COPY --from=build /app/bin/nitric /usr/bin/
