# Makefile Usage

The `mcpd` project includes a `Makefile` to streamline common developer tasks. 

!!! warning "Running make"
    All commands should be run from the **root of the repository**.

---

## Commands

!!! note "Environment"
    Most commands assume you have Go installed and available in your `PATH`.

### 🧱 Build Commands

- **Build the binary**
    ```bash
    make build
    ```

- **Remove the compiled binary from the working directory**
    ```bash
    make clean
    ```

- **Install the binary to your system (typically `/usr/local/bin`)**
    ```bash
    sudo make install
    ```

- **Uninstall the binary**
    ```bash
    sudo make uninstall
    ```

---

### 🧪 Test Commands

- **Run all Go tests**
    ```bash
    make test
    ```

---

## 📝 Documentation Commands

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

| Target         | Description                                       |
|----------------|---------------------------------------------------|
| `build`        | Compile the Go binary                             |
| `install`      | Install binary to system path                     |
| `uninstall`    | Remove installed binary                           |
| `clean`        | Remove compiled binary from working directory     |
| `test`         | Run all Go tests                                  |
| `docs-cli`     | Generate Markdown CLI reference docs              |
| `docs-nav`     | Update CLI doc nav in `mkdocs.yml`                |
| `docs-local`   | Serve docs locally via `mkdocs serve`             |
| `docs`         | Alias for `docs-local` (runs everything)          |

