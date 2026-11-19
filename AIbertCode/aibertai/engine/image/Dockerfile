FROM debian:bookworm-slim@sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tini \
 && rm -rf /var/lib/apt/lists/* \
 && groupadd --gid 1000 aibertai \
 && useradd --uid 1000 --gid 1000 --create-home --shell /bin/sh aibertai \
 && test -x /usr/bin/tini && test -x /bin/sleep
LABEL org.opencontainers.image.version="bookworm-1"
USER 1000:1000
WORKDIR /home/aibertai
