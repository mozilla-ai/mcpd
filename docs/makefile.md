# Makefile Usage

The `mcpd` project includes a `Makefile` to streamline common developer tasks. 

!!! warning "Running make"
    All commands should be run from the **root of the repository**.

---

## Commands

!!! note "Environment"
    Most commands assume you have Go installed and available in your `PATH`.

### 🧱 Build

- **Build the binary**
    ```bash
    make build
    ```

    !!! tip "Architectures and Operating Systems"
        You can explicitly build the binary for a different architecture (`amd64/arm64`) or operating systems with:
    
        * `make build-linux`
        * `make build-linux-arm64`

- **Remove the compiled binary from the working directory**
    ```bash
    make clean
    ```

- **Install the binary to your system (typically `/usr/local/bin`)**
    ```bash
    sudo make install
    ```

    !!! note "Dependency"
        The `install` target relies on the standard `build` target.


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

    !!! note "When to use"
        Run this command before submitting PRs that modify:

        * `internal/provider/mozilla_ai/data/registry.json`
        * `internal/provider/mozilla_ai/data/schema.json`

---

### 📜 License and Attribution

- **Check dependency licenses**
    ```bash
    make check-licenses
    ```

    !!! note "Allowed licenses"
        This validates that all dependencies use one of: `Apache-2.0`, `MIT`, `BSD-2-Clause`, `BSD-3-Clause`, `ZeroBSD`, or `Unlicense`.

- **Check NOTICE file is up to date**
    ```bash
    make check-notice
    ```

- **Generate NOTICE file**
    ```bash
    make notice
    ```

    !!! note "Third-party attribution"
        Regenerates the NOTICE file with current dependency license information.

---

### 🐳 Run

- **Start `mcpd` in a container**
    ```bash
    make local-up
    ```

    !!! warning "Default files"
        By default the following files will be mounted to the container:
        
        * `.mcpd.toml` - the project configuration file in this repository
        * `~/.config/mcpd/secrets.dev.toml` - the default location for runtime configuration

- **Stop mcpd**
    ```bash
    make local-down
    ```

---

### 📝 Documentation

These commands manage the [MkDocs](https://www.mkdocs.org) developer documentation site for `mcpd`.

!!! note "Environment"
    Docs commands assume you have `uv` installed and available in your `PATH` (in additon to Go).

- **Generate CLI reference docs from the Cobra commands**
    ```bash
    make docs-cli
    ```

- **Update `mkdocs.yaml` navigation for the CLI commands**
    ```bash
    make docs-nav
    ```

- **Serve the docs locally using MkDocs + uv: generate CLI docs, update nav, serve locally**
    ```bash
    make docs
    ```

    !!! tip "First time?"
        The `docs` command will create a virtual environment using `uv`, install MkDocs + Material theme, and start the local server at [http://localhost:8000/mcpd/](http://localhost:8000/mcpd/).

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
| `docs`              | Serve docs locally via `mkdocs serve`                    |
| `docs-local`        | Serve docs locally via `mkdocs serve`                    |
| `docs-nav`          | Update CLI doc nav in `mkdocs.yaml`                      |
| `install`           | Install binary to system path                            |
| `lint`              | Run linter with auto-fix (includes check-notice)         |
| `local-down`        | Stop a running `mcpd` Docker container                   |
| `local-up`          | Start `mcpd` in a Docker container                       |
| `notice`            | Generate NOTICE file with dependency licenses            |
| `test`              | Run all Go tests (includes lint)                         |
| `uninstall`         | Remove installed binary                                  |
| `validate-registry` | Validate Mozilla AI registry JSON schema                 |

