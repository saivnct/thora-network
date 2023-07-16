# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

# Build Geth in a stock Go builder container
FROM golang:1.20-alpine as builder

RUN apk add --no-cache gcc musl-dev linux-headers git

# Get dependencies - will also be cached if we won't change go.mod/go.sum
COPY go.mod /thora-network/
COPY go.sum /thora-network/
RUN cd /thora-network && go mod download

ADD . /thora-network
RUN cd /thora-network && go run build/ci.go install -static ./cmd/thora

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /thora-network/build/bin/thora /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["thora"]

# Add some metadata labels to help programatic image consumption
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
