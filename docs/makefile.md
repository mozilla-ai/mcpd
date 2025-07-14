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

- **Generate CLI reference docs from the Cobra commands**
    ```bash
    make docs-cli
    ```

- **Update `mkdocs.yaml` navigation for the CLI commands**
    ```bash
    make docs-nav
    ```

- **Serve the docs locally using MkDocs + uv**
    ```bash
    make docs-local
    ```

- **Full pipeline: generate CLI docs, update nav, serve locally**
    ```bash
    make docs
    ```

    !!! tip "First time?"
        The `docs-local` command will create a virtual environment using `uv`, install MkDocs + Material theme, and start the local server at [http://localhost:8000](http://localhost:8000).

---

## 🧭 Target Reference

Here’s a complete list of Makefile targets:

| Target              | Description                                   |
|---------------------|-----------------------------------------------|
| `build`             | Compile the Go binary                         |
| `build-linux`       | Compile the Go binary for Linux on amd64      |
| `build-linux-arm64` | Compile the Go binary for Linux on arm64      |
| `install`           | Install binary to system path                 |
| `uninstall`         | Remove installed binary                       |
| `clean`             | Remove compiled binary from working directory |
| `test`              | Run all Go tests                              |
| `local-up`          | Start `mcpd` in a Docker container            |
| `local-down`        | Stop a running `mcpd` Docker container        |
| `docs-cli`          | Generate Markdown CLI reference docs          |
| `docs-nav`          | Update CLI doc nav in `mkdocs.yaml`           |
| `docs-local`        | Serve docs locally via `mkdocs serve`         |
| `docs`              | Alias for `docs-local` (runs everything)      |

