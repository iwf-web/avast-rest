ARG GO_VERSION=1.26.1
FROM --platform=linux/amd64 golang:${GO_VERSION}-alpine AS build

RUN apk update && apk upgrade

WORKDIR /go/src
ADD . /go/src/av-rest/
RUN cd /go/src/av-rest && go mod tidy && go build -v

# ── Runtime image ────────────────────────────────────────────────────────────
FROM --platform=linux/amd64 debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        gnupg \
        tzdata

RUN DIST=$(. /etc/os-release; echo "$ID-$VERSION_CODENAME") && echo "deb https://repo.avcdn.net/linux-av/deb $DIST release" > /etc/apt/sources.list.d/avast.list

RUN curl -fsSL https://repo.avcdn.net/linux-av/doc/avast-gpg-key.asc -o /etc/apt/trusted.gpg.d/avast-gpg-key.asc
RUN apt-get update
RUN apt-get install -y --no-install-recommends \
        avast \
        avast-license
RUN rm -rf /var/lib/apt/lists/*

# Set timezone
ENV TZ=Europe/Zurich

# Set up directories the avast user needs to own at runtime.
# avast package creates the avast user; /var/run/avast is not created by the
# package because it's normally managed by systemd.
RUN mkdir -p /etc/avast /var/run/avast && \
    chown -R avast:avast /etc/avast /var/run/avast

# Copy compiled binary
COPY --from=build /go/src/av-rest/av-rest /usr/bin/av-rest

COPY entrypoint.sh /usr/bin/entrypoint.sh
RUN chmod +x /usr/bin/entrypoint.sh

# Environment – Avast paths (override if your installation differs)
ENV SCANNER=avast
ENV AVAST_SCAN_BIN=/usr/bin/scan
ENV AVAST_VDF_DIR=/var/lib/avast

ENV PORT=9000
ENV HEALTHCHECK_MAX_SIGNATURE_AGE=48

USER avast
ENTRYPOINT ["/usr/bin/entrypoint.sh"]
