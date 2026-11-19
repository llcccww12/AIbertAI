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
ENTRYPOINT ["aibertai-server"]
