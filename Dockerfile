

FROM chainguard/wolfi-base:latest

RUN apk add --no-cache tini

# Include curl in the final image.
RUN set -ex \
    && apk update \
    && apk add --no-cache curl tini wireguard-tools iputils iproute2 \
    && rm -rf /var/cache/apk/*  \
    && rm -rf /tmp/*

STOPSIGNAL SIGTERM

# Get the boot-script-service from the builder stage.
COPY cloud-init-server /usr/local/bin/

ENV TOKEN_URL="http://opaal:3333/token"
ENV SMD_URL="http://smd:27779"
ENV LISTEN_ADDR="0.0.0.0:27777"


# nobody 65534:65534
USER 65534:65534

# Set up the command to start the service.
CMD /usr/local/bin/cloud-init-server 

ENTRYPOINT ["/sbin/tini", "--"]
