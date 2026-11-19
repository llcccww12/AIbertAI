FROM golang:1.27.0-bookworm AS host-build
RUN apt-get update \
 && apt-get install -y --no-install-recommends python3 \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /engine
COPY engine/go.mod engine/go.sum ./
RUN go mod download
COPY engine/ ./
ARG ENGINE_BUILD_VERSION
RUN python3 scripts/package-release.py --output /host-releases --version "$ENGINE_BUILD_VERSION"

FROM scratch AS host-artifacts
COPY --from=host-build /host-releases/ /

FROM golang:1.27.0-bookworm AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -buildvcs=false -o /out/aibertai-server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates poppler-utils \
 && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/aibertai-server /usr/local/bin/aibertai-server
COPY --from=host-artifacts / /opt/aibertai/host-releases/
ENV AIBERTAI_ENGINE_RELEASE_DIR=/opt/aibertai/host-releases
ENTRYPOINT ["aibertai-server"]
