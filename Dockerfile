# syntax=docker/dockerfile:1.7
FROM ubuntu:24.04 AS duckdb-ext
ARG TARGETARCH
ARG DUCKDB_VERSION="1.1.3"

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl unzip \
 && rm -rf /var/lib/apt/lists/*

# Map TARGETARCH -> DuckDB CLI artifact name
# amd64 => linux-amd64, arm64 => linux-aarch64
RUN case "${TARGETARCH}" in \
      amd64)  CLI_URL="https://github.com/duckdb/duckdb/releases/download/v${DUCKDB_VERSION}/duckdb_cli-linux-amd64.zip" ;; \
      arm64)  CLI_URL="https://github.com/duckdb/duckdb/releases/download/v${DUCKDB_VERSION}/duckdb_cli-linux-aarch64.zip" ;; \
      *) echo "unsupported arch: ${TARGETARCH}" && exit 1 ;; \
    esac \
 && curl -fsSL "$CLI_URL" -o /tmp/duckdb.zip \
 && unzip /tmp/duckdb.zip -d /usr/local/bin \
 && chmod +x /usr/local/bin/duckdb \
 && rm -f /tmp/duckdb.zip

ENV DUCKDB_HOME=/duckdb_home
RUN mkdir -p "$DUCKDB_HOME"

# Preinstall whatever you need
RUN duckdb -c "INSTALL 'json';" \
 && duckdb -c "INSTALL 'parquet';" 

# -------- runtime image --------
FROM ubuntu:24.04
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates wireguard-tools tini \
 && rm -rf /var/lib/apt/lists/*

COPY cloud-init-server /usr/local/bin/cloud-init-server

ENV DUCKDB_HOME=/opt/duckdbhome
RUN mkdir -p "$DUCKDB_HOME" && chown 65534:65534 "$DUCKDB_HOME"

# Copy the preinstalled extensions (built for the current arch)
COPY --from=duckdb-ext /duckdb_home/ ${DUCKDB_HOME}/
RUN chown -R 65534:65534 ${DUCKDB_HOME}

ENV TOKEN_URL="http://opaal:3333/token"
ENV SMD_URL="http://smd:27779"
ENV LISTEN="0.0.0.0:27777"

USER 65534:65534
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/usr/local/bin/cloud-init-server"]

