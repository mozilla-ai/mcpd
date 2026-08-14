# Makefile Usage

The `mcpd` project includes a `Makefile` to streamline common developer tasks. 

{% hint style="warning" %}
**Running make**

All commands should be run from the **root of the repository**.
{% endhint %}

---

## Commands

{% hint style="info" %}
**Environment**

Most commands assume you have Go installed and available in your `PATH`.
{% endhint %}

### 🧱 Build

- **Build the binary**
    ```bash
    make build
    ```

    {% hint style="success" %}
    **Architectures and Operating Systems**

    You can explicitly build the binary for a different architecture (`amd64/arm64`) or operating systems with:

    * `make build-linux`
    * `make build-linux-arm64`
    {% endhint %}

- **Remove the compiled binary from the working directory**
    ```bash
    make clean
    ```

- **Install the binary to your system (typically `/usr/local/bin`)**
    ```bash
    sudo make install
    ```

    {% hint style="info" %}
    **Dependency**

    The `install` target relies on the standard `build` target.
    {% endhint %}


- **Uninstall the binary**
    ```bash
    sudo make uninstall
    ```

---

### 🧪 Test

- **Run all Go tests**
    ```bash
    make test
    ```

---

### ✅ Validation

- **Run linter with auto-fix**
    ```bash
    make lint
    ```

- **Validate Mozilla AI registry against JSON schema**
    ```bash
    make validate-registry
    ```

    {% hint style="info" %}
    **When to use**

    Run this command before submitting PRs that modify:

    * `internal/provider/mozilla_ai/data/registry.json`
    * `internal/provider/mozilla_ai/data/schema.json`
    {% endhint %}

---

### 📜 License and Attribution

- **Check dependency licenses**
    ```bash
    make check-licenses
    ```

    {% hint style="info" %}
    **Allowed licenses**

    This validates that all dependencies use one of: `Apache-2.0`, `MIT`, `BSD-2-Clause`, `BSD-3-Clause`, `ZeroBSD`, or `Unlicense`.
    {% endhint %}

- **Check NOTICE file is up to date**
    ```bash
    make check-notice
    ```

- **Generate NOTICE file**
    ```bash
    make notice
    ```

    {% hint style="info" %}
    **Third-party attribution**

    Regenerates the NOTICE file with current dependency license information.
    {% endhint %}

---

### 🐳 Run

- **Start `mcpd` in a container**
    ```bash
    make local-up
    ```

    {% hint style="warning" %}
    **Default files**

    By default the following files will be mounted to the container:

    * `.mcpd.toml` - the project configuration file in this repository
    * `~/.config/mcpd/secrets.dev.toml` - the default location for runtime configuration
    {% endhint %}

- **Stop mcpd**
    ```bash
    make local-down
    ```

---

### 📝 Documentation

These commands build the [GitBook](https://docs.mozilla.ai/mcpd) documentation site for `mcpd`.

{% hint style="info" %}
**Environment**

Docs commands assume you have `uv` installed and available in your `PATH` (in addition to Go).
{% endhint %}

- **Generate CLI reference docs from the Cobra commands**
    ```bash
    make docs-cli
    ```

- **Generate the OpenAPI specification**
    ```bash
    make docs-api
    ```

- **Build the GitBook site into `site/` (runs the generators first)**
    ```bash
    make docs
    ```

    {% hint style="info" %}
    **Previewing**

    GitBook renders the published site from the `gitbook-docs` branch, so there is no local dev server. `make docs` assembles the publishable Markdown into `site/` for inspection.
    {% endhint %}

---

## 🧭 Target Reference

Here's a complete list of Makefile targets:

| Target              | Description                                              |
|---------------------|----------------------------------------------------------|
| `build`             | Compile the Go binary                                    |
| `build-dev`         | Compile the Go binary for development (no optimizations) |
| `build-linux`       | Compile the Go binary for Linux on amd64                 |
| `build-linux-arm64` | Compile the Go binary for Linux on arm64                 |
| `check-licenses`    | Validate all dependency licenses are allowed             |
| `check-notice`      | Verify NOTICE file is up to date                         |
| `clean`             | Remove compiled binary from working directory            |
| `docs`              | Build the GitBook documentation site into `site/`        |
| `docs-api`          | Generate the OpenAPI specification                       |
| `docs-cli`          | Generate CLI reference docs from the Cobra commands      |
| `install`           | Install binary to system path                            |
| `lint`              | Run linter with auto-fix (includes check-notice)         |
| `local-down`        | Stop a running `mcpd` Docker container                   |
| `local-up`          | Start `mcpd` in a Docker container                       |
| `notice`            | Generate NOTICE file with dependency licenses            |
| `test`              | Run all Go tests (includes lint)                         |
| `uninstall`         | Remove installed binary                                  |
| `validate-registry` | Validate Mozilla AI registry JSON schema                 |

