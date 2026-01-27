# Installing `mcpd`

## via Homebrew

Add the Mozilla.ai tap:

```bash
brew tap mozilla-ai/tap
```

Then install `mcpd`:

```bash
brew install mcpd
```

Or install directly from the cask:

```bash
brew install --cask mozilla-ai/tap/mcpd
```

## via GitHub releases

Official releases can be found on the [mcpd GitHub releases page](https://github.com/mozilla-ai/mcpd/releases).

The following is an example of manually downloading and installing `mcpd` using `curl` and `jq` by running `install_mcpd`:

```bash
function install_mcpd() {
    command -v curl >/dev/null || { echo "curl not found"; return 1; }
    command -v jq >/dev/null || { echo "jq not found"; return 1; }

    latest_version=$(curl -s https://api.github.com/repos/mozilla-ai/mcpd/releases/latest | jq -r .tag_name)
    os=$(uname)
    arch=$(uname -m)

    zip_name="mcpd_${os}_${arch}.tar.gz"
    url="https://github.com/mozilla-ai/mcpd/releases/download/${latest_version}/${zip_name}"

    echo "Downloading: $url"
    curl -sSL "$url" -o "$zip_name" || { echo "Download failed"; return 1; }

    echo "Extracting: $zip_name"
    tar -xzf "$zip_name" mcpd || { echo "Extraction failed"; return 1; }

    echo "Installing to /usr/local/bin"
    sudo mv mcpd /usr/local/bin/mcpd && sudo chmod +x /usr/local/bin/mcpd || { echo "Install failed"; return 1; }

    rm -f "$zip_name"
    echo "mcpd installed successfully"
}
```

!!! info "macOS Gatekeeper quarantine"
    If you're on macOS, remove the quarantine flag before running `mcpd`:
    ```
    xattr -d com.apple.quarantine mcpd
    ```

## via local Go binary build

```bash
# Clone the Git repo
git clone git@github.com:mozilla-ai/mcpd.git
cd mcpd
# Checkout a specific tag (or build latest main)
git fetch --tags
git checkout v0.0.5
# Use Makefile commands to build and install mcpd
make build
sudo make install # Installs mcpd 'globally' to /usr/local/bin
```

## Run with Docker

`mcpd` is available as the Docker image [mzdotai/mcpd](https://hub.docker.com/repository/docker/mzdotai/mcpd/general).

!!! note "Dockerfile environment variables"
    The [Dockerfile](https://github.com/mozilla-ai/mcpd/blob/main/Dockerfile) defines sensible defaults for configuration via environment variables. These can be overridden at runtime using `docker run -e KEY=VALUE`.

### Default environment variables

| Name                | Default Value                               |
|---------------------|---------------------------------------------|
| `MCPD_API_PORT`     | `8090`                                      |
| `MCPD_LOG_LEVEL`    | `info`                                      |
| `MCPD_LOG_PATH`     | `/var/log/mcpd/mcpd.log`                    |
| `MCPD_CONFIG_FILE`  | `/etc/mcpd/.mcpd.toml`                      |
| `MCPD_RUNTIME_FILE` | `/home/mcpd/.config/mcpd/secrets.prod.toml` |


To run `mcpd` with Docker, map the required port and bind mount your `.mcpd.toml` configuration file and runtime secrets file:

```bash
docker run  -p 8090:8090 \
            -v $PWD/.mcpd.toml:/etc/mcpd/.mcpd.toml \
            -v $HOME/.config/mcpd/secrets.dev.toml:/home/mcpd/.config/mcpd/secrets.prod.toml \
            -e MCPD_LOG_LEVEL=debug \
            mzdotai/mcpd:v0.0.5
```

### Running Docker-based MCP servers from containerized `mcpd`

If your MCP servers use the Docker runtime, mount the host's Docker socket to allow mcpd to manage containers on the host:

```bash
docker run  -p 8090:8090 \
            -v /var/run/docker.sock:/var/run/docker.sock \
            -v $PWD/.mcpd.toml:/etc/mcpd/.mcpd.toml \
            -v $HOME/.config/mcpd/secrets.dev.toml:/home/mcpd/.config/mcpd/secrets.prod.toml \
            -e MCPD_LOG_LEVEL=debug \
            mzdotai/mcpd:v0.0.5
```

!!! warning "Security Note"
    Mounting the Docker socket grants the container full access to the host's Docker daemon. Only use this with trusted images.

### CI/CD Deployment (GitHub Actions)

For automated deployments, a reference GitHub Actions workflow is available in the repository that demonstrates:

- Version-pinned deployments using release tags
- Secret resolution from GitHub Secrets into a `.env` file for Docker
- Configuration validation

See the following files:

- [`.github/workflows/deploy.yaml`](https://github.com/mozilla-ai/mcpd/blob/main/.github/workflows/deploy.yaml) - Example workflow
- [`scripts/resolve-secrets.sh`](https://github.com/mozilla-ai/mcpd/blob/main/scripts/resolve-secrets.sh) - Resolves `MCPD__` prefixed secrets

Use `mcpd config export` to generate the portable execution context required for deployment.
