# syntax=docker/dockerfile:1

# ==============================================================================
# Final Stage: Build the production image.
# Includes NodeJS to give mcpd access to the npx binary.
# ==============================================================================
FROM node:24.5.0-alpine3.22

# --- Metadata ---
LABEL org.opencontainers.image.authors="Mozilla AI <security@mozilla.ai>"
LABEL org.opencontainers.image.description="A container for the mcpd application."
# The version label should be dynamically overridden in a CI/CD pipeline
LABEL org.opencontainers.image.version="dev"

ARG MCPD_USER=mcpd
ARG MCPD_HOME=/home/$MCPD_USER

# Sensible defaults but can be easily overridden by the user with `docker run -e KEY=VALUE`.
ENV MCPD_API_PORT=8090
ENV MCPD_LOG_LEVEL=info
ENV MCPD_LOG_PATH=/var/log/mcpd/mcpd.log
ENV MCPD_CONFIG_FILE=/etc/mcpd/.mcpd.toml
ENV MCPD_RUNTIME_FILE=/home/mcpd/.config/mcpd/secrets.prd.toml

USER root

# Installs python, pip and tools
RUN apk add --no-cache \
    python3=3.12.11-r0 \
    py3-pip=25.1.1-r0 \
    py3-setuptools=80.9.0-r0 \
    py3-wheel=0.46.1-r0

# Installs 'tini', a lightweight init system to properly manage processes.
RUN apk add --no-cache tini=0.19.0-r3

#  - Adds a dedicated non-root group and user for security (using the ARG).
#  - Creates necessary directories for configs, logs, and user data.
#  - Sets correct ownership for the non-root user.
RUN addgroup -S $MCPD_USER && \
    adduser -D -S -h $MCPD_HOME -G $MCPD_USER $MCPD_USER && \
    mkdir -p \
      $MCPD_HOME/.config/mcpd \
      /var/log/mcpd \
      /etc/mcpd && \
    chown -R $MCPD_USER:$MCPD_USER $MCPD_HOME /var/log/mcpd

# Copy uv/uvx binaries from image.
COPY --from=ghcr.io/astral-sh/uv:0.8.4 /uv /uvx /usr/local/bin/

# Copy application binary and set ownership to the non-root user.
# IMPORTANT: Config/secrets are NOT copied. They should be mounted at runtime.
COPY --chown=$MCPD_USER:$MCPD_USER mcpd /usr/local/bin/mcpd

# Switch to the non-root user before execution.
USER $MCPD_USER
WORKDIR $MCPD_HOME

EXPOSE $MCPD_API_PORT

# Use 'tini' as the entrypoint to properly handle process signals (like CTRL+C)
# and prevent zombie processes, ensuring clean container shutdown.
ENTRYPOINT ["/sbin/tini", "--"]

CMD mcpd daemon \
    --addr 0.0.0.0:$MCPD_API_PORT \
    --log-level $MCPD_LOG_LEVEL \
    --log-path $MCPD_LOG_PATH \
    --config-file $MCPD_CONFIG_FILE \
    --runtime-file $MCPD_RUNTIME_FILE

# Example run:
# docker run -p 8090:8090 \
#            -v $PWD/.mcpd.toml:/etc/mcpd/.mcpd.toml \
#            -v $HOME/.config/mcpd/secrets.dev.toml:/home/mcpd/.config/mcpd/secrets.prd.toml \
#            -e MCPD_LOG_LEVEL=debug \
#            mzdotai/mcpd:v0.0.4