FROM ubuntu:24.04

# Optional: slim this down a bit
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wireguard-tools tini\
  && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY cloud-init-server /usr/local/bin/cloud-init-server

# Configuration via environment variables
ENV TOKEN_URL="http://opaal:3333/token"
ENV SMD_URL="http://smd:27779"
ENV LISTEN="0.0.0.0:27777"

# Set non-root user
USER 65534:65534

# Tini for proper signal forwarding
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/usr/local/bin/cloud-init-server"]
