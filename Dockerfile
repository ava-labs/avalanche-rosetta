# ------------------------------------------------------------------------------
# Build avalanche rosetta
# ------------------------------------------------------------------------------
FROM golang:1.15 AS build

RUN git clone https://github.com/figment-networks/avalanche-rosetta.git \
  /go/src/github.com/figment-networks/avalanche-rosetta

WORKDIR /go/src/github.com/figment-networks/avalanche-rosetta

ENV CGO_ENABLED=1
ENV GOARCH=amd64
ENV GOOS=linux

RUN go mod download

RUN \
  GO_VERSION=$(go version | awk {'print $3'}) \
  GIT_COMMIT=$(git rev-parse HEAD) \
  make build

# ------------------------------------------------------------------------------
# Target container for running the node and rosetta server
# ------------------------------------------------------------------------------
FROM ubuntu:18.04

ENV AVALANCHE_VERSION=v1.0.5

# Install dependencies
RUN apt-get update -y && \
    apt-get install -y wget

WORKDIR /app

# Install avalanchego
RUN \
  wget -q https://github.com/ava-labs/avalanchego/releases/download/$AVALANCHE_VERSION/avalanchego-linux-$AVALANCHE_VERSION.tar.gz && \
  tar -xzf avalanchego-linux-$AVALANCHE_VERSION.tar.gz && \
  rm *.gz && \
  mv avalanchego-$AVALANCHE_VERSION/* .

# Install rosetta server
COPY --from=build \
  /go/src/github.com/figment-networks/avalanche-rosetta/avalanche-rosetta \
  /app/rosetta

COPY --from=build \
  /go/src/github.com/figment-networks/avalanche-rosetta/docker/start.sh \
  /app/start

EXPOSE 9650
EXPOSE 8081

CMD ["/app/start"]
