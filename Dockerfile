# syntax=docker/dockerfile:1

# ==============================================================================
# Builder Stage: Fetch uv binaries
# ==============================================================================
FROM ghcr.io/astral-sh/uv:0.7.20 AS uv-builder

# ==============================================================================
# Final Stage: Build the production image.
# Includes NodeJS to give mcpd access to the npx binary.
# ==============================================================================
FROM node:current-alpine3.22

ARG MCPD_VERSION=unknown

# --- Metadata ---
# The version label should be dynamically overridden in a CI/CD pipeline
# (e.g., --label "org.opencontainers.image.version=${GIT_TAG}").
LABEL org.opencontainers.image.authors="Mozilla AI <security@mozilla.ai>"
LABEL org.opencontainers.image.description="A container for the mcpd application."
LABEL org.opencontainers.image.version=$MCPD_VERSION

ARG MCPD_USER=mcpd
ARG MCPD_HOME=/home/$MCPD_USER

# Sensible defaults but can be easily overridden by the user with `docker run -e KEY=VALUE`.
ENV MCPD_API_PORT=8090
ENV MCPD_LOG_LEVEL=info

#  - Installs 'tini', a lightweight init system to properly manage processes.
#  - Adds a dedicated non-root group and user for security (using the ARG).
#  - Creates necessary directories for configs, logs, and user data.
#  - Sets correct ownership for the non-root user.
USER root
RUN apk add --no-cache python3 py3-pip tini && \
    addgroup -S $MCPD_USER && \
    adduser -D -S -h $MCPD_HOME -G $MCPD_USER $MCPD_USER && \
    mkdir -p \
      $MCPD_HOME/.config/mcpd \
      /var/log/mcpd \
      /etc/mcpd && \
    chown -R $MCPD_USER:$MCPD_USER $MCPD_HOME /var/log/mcpd

# Copy binaries from the dedicated 'uv-builder' stage.
COPY --from=uv-builder /uv /uvx /usr/local/bin/

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
    --log-path /var/log/mcpd/mcpd.log \
    --config-file /etc/mcpd/.mcpd.toml \
    --runtime-file /home/mcpd/.config/mcpd/secrets.prd.toml