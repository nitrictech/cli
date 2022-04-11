FROM python:3.9-slim

LABEL "repository"="https://github.com/nitrictech/cli"
LABEL "homepage"="https://nitric.io"
LABEL org.opencontainers.image.description="The Nitric CLI, in a Docker container."

ENV GOLANG_VERSION 1.17.3
ENV GOLANG_SHA256 550f9845451c0c94be679faf116291e7807a8d78b43149f9506c1b15eb89008c

# Install deps all in one step
RUN apt-get update -y && \
  apt-get install -y \
  apt-transport-https \
  build-essential \
  ca-certificates \
  net-tools \
  curl \
  git \
  gnupg \
  software-properties-common \
  wget \
  unzip && \
  # Get all of the signatures we need all at once.
  curl -fsSL https://deb.nodesource.com/gpgkey/nodesource.gpg.key  | apt-key add - && \
  curl -fsSL https://dl.yarnpkg.com/debian/pubkey.gpg              | apt-key add - && \
  curl -fsSL https://download.docker.com/linux/debian/gpg          | apt-key add - && \
  curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
  curl -fsSL https://packages.microsoft.com/keys/microsoft.asc     | apt-key add - && \
  # IAM Authenticator for EKS
  curl -fsSLo /usr/bin/aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-07-26/bin/linux/amd64/aws-iam-authenticator && \
  chmod +x /usr/bin/aws-iam-authenticator && \
  # AWS v2 cli
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
  unzip awscliv2.zip && \
  ./aws/install && \
  rm -rf aws && \
  # Add additional apt repos all at once
  echo "deb https://deb.nodesource.com/node_14.x $(lsb_release -cs) main"                         | tee /etc/apt/sources.list.d/node.list             && \
  echo "deb https://dl.yarnpkg.com/debian/ stable main"                                           | tee /etc/apt/sources.list.d/yarn.list             && \
  echo "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"      | tee /etc/apt/sources.list.d/docker.list           && \
  echo "deb http://packages.cloud.google.com/apt cloud-sdk-$(lsb_release -cs) main"               | tee /etc/apt/sources.list.d/google-cloud-sdk.list && \
  echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/azure.list            && \
  # Install second wave of dependencies
  apt-get update -y && \
  apt-get install -y \
  azure-cli \
  docker-ce \
  google-cloud-sdk \
  nodejs \
  yarn && \
  # Clean up the lists work
  rm -rf /var/lib/apt/lists/*

# Install Go
RUN curl -fsSLo /tmp/go.tgz https://golang.org/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz; \
  echo "${GOLANG_SHA256} /tmp/go.tgz" | sha256sum -c -; \
  tar -C /usr/local -xzf /tmp/go.tgz; \
  rm /tmp/go.tgz; \
  export PATH="/usr/local/go/bin:$PATH"; \
  go version
ENV GOPATH /workspace/go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

# Passing --build-arg PULUMI_VERSION=vX.Y.Z will use that version
# of the SDK. Otherwise, we use whatever get.pulumi.com thinks is
# the latest
ARG PULUMI_VERSION

# Install the Pulumi SDK, including the CLI and language runtimes.
RUN curl -fsSL https://get.pulumi.com/ | bash -s -- --version $PULUMI_VERSION && \
  mv ~/.pulumi/bin/* /usr/bin

RUN pulumi plugin install resource gcp
RUN pulumi plugin install resource random

ENV HOST_DOCKER_INTERNAL_IFACE eth0
ENV PULUMI_SKIP_UPDATE_CHECK "true"

COPY bin/nitric /usr/bin/
