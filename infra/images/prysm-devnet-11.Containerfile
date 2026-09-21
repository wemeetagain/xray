ARG GO_IMAGE=docker.io/library/golang:1.26-bookworm
FROM ${GO_IMAGE} AS build

WORKDIR /src
COPY prysm /src/prysm
COPY xray /src/xray
# The xray probe module is bridged in as github.com/ethp2p/xray via a go workspace, so
# prysm resolves probe.Wrap from the local module instead of the network.
# The two go get steps repair the workspace module graph: prysm still requires the ancient
# monolithic google.golang.org/genproto, which collides with the split genproto submodules
# that its grpc/otel dependencies pull in (ambiguous import for googleapis/api + rpc).
RUN go work init /src/prysm /src/xray \
    && cd /src/prysm \
    && go get google.golang.org/genproto@v0.0.0-20260918162117-cecb64721679 \
    && go get google.golang.org/genproto/googleapis/api@v0.0.0-20260917231906-eeb232e0883d \
              google.golang.org/genproto/googleapis/rpc@v0.0.0-20260917231906-eeb232e0883d \
    && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/beacon-chain ./cmd/beacon-chain

FROM docker.io/library/debian:bookworm-slim

RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 1000 ethp2p \
    && useradd --uid 1000 --gid 1000 --no-create-home --shell /usr/sbin/nologin ethp2p

COPY --from=build /out/beacon-chain /usr/local/bin/beacon-chain

ENV HOME=/tmp
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/beacon-chain"]
