FROM rockylinux:9 AS builder

RUN dnf update -y && \
    dnf install -y epel-release && \
    dnf install -y tini wireguard-tools && \
    dnf clean all

FROM registry.access.redhat.com/ubi9/ubi-minimal

# Copy the wg binary from the builder image
COPY --from=builder /usr/bin/wg /usr/bin/wg
COPY --from=builder /usr/bin/wg-quick /usr/bin/wg-quick
COPY --from=builder /usr/bin/tini /usr/bin/tini

# Install dnf and certificates
RUN microdnf install -y dnf ca-certificates && dnf clean all


RUN microdnf install -y \
      iproute \
      iputils \
      libstdc++ \
      libgcc && \
    microdnf clean all

# Copy binary from builder
COPY cloud-init-server /usr/local/bin/cloud-init-server

# Configuration via environment variables
ENV TOKEN_URL="http://opaal:3333/token"
ENV SMD_URL="http://smd:27779"
ENV LISTEN="0.0.0.0:27777"

# Set non-root user
# UID 65534 is typically "nobody" on Red Hat systems
USER 65534:65534

# Tini for proper signal forwarding
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/usr/local/bin/cloud-init-server"]
