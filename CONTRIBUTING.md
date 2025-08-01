# Contributing to `mcpd`

Thank you for your interest in contributing to the `mcpd` project!
We welcome contributions from everyone and are grateful for your help in making this project better.

By contributing to this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## How to Contribute

We encourage contributions via GitHub pull requests. Before you start, please review our [documented requirements](https://mozilla-ai.github.io/mcpd/requirements/).

### Reporting Bugs & Suggesting Features

* **Bugs:** Please use our [Bug Report template](.github/ISSUE_TEMPLATE/bug_report.yaml) to provide clear steps to reproduce and environment details.
* **Features:** Please use our [Feature Request template](.github/ISSUE_TEMPLATE/feature_request.yaml) to describe the problem your idea solves and your proposed solution.
* **Search First:** Before creating a new issue, please search existing issues to see if your topic has already been discussed.

### Requirements

* [gofumpt](https://github.com/mvdan/gofumpt)
* [golangci-lint](https://golangci-lint.run/welcome/install/)

### Contributing Code

1. **Fork** the repository on GitHub.
2. **Clone** your forked repository to your local machine.
    ```bash
    git clone https://github.com/{YOUR_GITHUB_USERNAME}/mcpd.git
    cd mcpd
    ```
3. **Create a new branch** for your changes based on the `main` branch.
    ```bash
    git checkout main
    git pull origin main
    git checkout -b your-feature-or-bugfix-branch
    ```
4. **Make your changes:** You can build your changes with:
   ```bash
   make build
   ```
5. **Format and Lint:** Ensure your code passes linting and formatting checks:
   ```bash
   # Run golangci-lint (uses .golangci.yaml config)
   golangci-lint run
   
   # Auto-fix formatting issues (recommended)
   golangci-lint run --fix
   ```
6. **Add Unit Tests:** All new features and bug fixes should be accompanied by relevant unit tests. Run tests with:
   ```bash
   make test
   ```
7. **Commit your changes** with a clear and descriptive message.
8. **Push your branch** to your forked repository.
9. **Open a Pull Request** from your branch to the `main` branch of the upstream `mozilla-ai/mcpd` repository, 
  reference the relevant GitHub issue in your PR summary.

## Security Vulnerabilities

If you discover a security vulnerability, please **DO NOT** open a public issue. Report it responsibly by following our [Security Policy](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed as described in [LICENSE](LICENSE.md).