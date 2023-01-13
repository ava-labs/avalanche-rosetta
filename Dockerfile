# ------------------------------------------------------------------------------
# Build camino
# ------------------------------------------------------------------------------
FROM golang:1.19.1 AS camino

ARG CAMINO_VERSION

RUN git clone https://github.com/chain4travel/camino-node.git \
  /go/src/github.com/chain4travel/camino-node

WORKDIR /go/src/github.com/chain4travel/camino-node

RUN git checkout $CAMINO_VERSION && \
    ./scripts/build.sh

# ------------------------------------------------------------------------------
# Build camino rosetta
# ------------------------------------------------------------------------------
FROM golang:1.19.1 AS rosetta

ARG ROSETTA_VERSION

# RUN git clone https://github.com/chain4travel/camino-rosetta.git \
#   /go/src/github.com/chain4travel/camino-rosetta

COPY . /go/src/github.com/chain4travel/camino-rosetta

WORKDIR /go/src/github.com/chain4travel/camino-rosetta

ENV CGO_ENABLED=1
ENV GOARCH=amd64
ENV GOOS=linux

RUN git checkout $ROSETTA_VERSION && \
    go mod download

RUN \
  GO_VERSION=$(go version | awk {'print $3'}) \
  GIT_COMMIT=$(git rev-parse HEAD) \
  make build

# ------------------------------------------------------------------------------
# Target container for running the node and rosetta server
# ------------------------------------------------------------------------------
FROM ubuntu:20.04

# Install dependencies
RUN apt-get update -y && \
    apt-get install -y wget

WORKDIR /app

# Install camino daemon
COPY --from=camino \
  /go/src/github.com/chain4travel/camino-node/build/camino-node \
  /app/camino-node

# Install evm plugin
COPY --from=camino \
  /go/src/github.com/chain4travel/camino-node/build/plugins \
  /app/plugins

# Install rosetta server
COPY --from=rosetta \
  /go/src/github.com/chain4travel/camino-rosetta/rosetta-server \
  /app/rosetta-server

# Install rosetta runner
COPY --from=rosetta \
  /go/src/github.com/chain4travel/camino-rosetta/rosetta-runner \
  /app/rosetta-runner

# Install service start script
COPY --from=rosetta \
  /go/src/github.com/chain4travel/camino-rosetta/docker/entrypoint.sh \
  /app/entrypoint.sh

EXPOSE 9650
EXPOSE 9651
EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
