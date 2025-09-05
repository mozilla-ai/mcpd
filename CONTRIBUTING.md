# Contributing to `mcpd`

Thank you for your interest in contributing to the `mcpd` project!
We welcome contributions from everyone and are grateful for your help in making this project better.

By contributing to this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

---

## **Guidelines for Contributions**

### Ground Rules

- Review issue discussion fully before starting work. Engage in the thread first when an issue is under discussion.
- PRs must build on agreed direction where ones exist. If there is no agreed direction, seek consensus from the core maintainers.
- PRs with "drive-by" unrelated changes or untested refactors will be closed.
- Untested or failing code is not eligible for review.
- PR description **must** follow the PR template and explain **what** changed, **why**, and **how to test**.
- Links to related issues are required.
- Duplicate PRs will be automatically closed.
- Only have 1-2 PRs open at a time. Any further PRs will be closed.

**Maintainers reserve the right to close issues and PRs that do not align with the library roadmap.**

### Code Clarity and Style

- **Readability first:** Code must be self-documenting—if it is not self-explanatory, it should include clear, concise comments where logic is non-obvious.
- **Consistent Style:** Follow existing codebase style (e.g., function naming, Go conventions)
- **No dead/debug code:** Remove commented-out blocks, leftover debug statements, unrelated refactors
- Failure modes must be documented and handled with robust error handling.

### Testing Requirements

- **Coverage:** All new functionality must include unit tests covering both happy paths and relevant edge cases.
- **Passing tests:** All linting and formatting checks must pass (see below on how to run).
- **No silent failures:** Tests should fail loudly on errors. No placeholder tests.

### Scope and Size

- **One purpose per PR:** No kitchen-sink PRs mixing bugfixes, refactors, and features.
- **Small, reviewable chunks:** If your PR is too large to review in under 30 minutes, break it up into chunks.
- Each chunk must be independently testable and reviewable
- If you can't explain why it can't be split, expect an automatic request for refactoring.
- Pull requests that are **large** (>500 LOC changed) or span multiple subsystems will be closed with automatic requests for refactoring.
- If the PR is to implement a new feature, please first make a GitHub issue to suggest the feature and allow for discussion. We reserve the right to close feature implementations and request discussion via an issue.

## How to Contribute

We encourage contributions via GitHub pull requests. Before you start, please review our [documented requirements](https://mozilla-ai.github.io/mcpd/requirements/).

### **Browse Existing Issues** 🔍
- Check the Issues page to see if there are any tasks you'd like to tackle.
- Look for issues labeled **`good first issue`** if you're new to the project—they're a great place to start.

### **Report Issues** 🐛
- **Bugs:** Please use our [Bug Report template](.github/ISSUE_TEMPLATE/bug_report.yaml) to provide clear steps to reproduce and environment details.
- **Search First:** Before creating a new issue, please search existing issues to see if your topic has already been discussed.
- Provide as much detail as possible, including the steps to reproduce the issue and expected vs. actual behavior.

### **Suggest Features** 🚀
- **Features:** Please use our [Feature Request template](.github/ISSUE_TEMPLATE/feature_request.yaml) to describe the problem your idea solves and your proposed solution.
- Share why the feature is important and any alternative solutions you've considered.
- If the PR is to implement a new feature, please first make a GitHub issue to suggest the feature and allow for discussion.

### Requirements

* [gofumpt](https://github.com/mvdan/gofumpt)
* [golangci-lint](https://golangci-lint.run/welcome/install/)

### **Submit Pull Requests** 💻

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
7. **Validate Registry (if applicable):** If you've modified the Mozilla AI registry, validate your changes:
   ```bash
   make validate-registry
   ```
   This applies to changes in:
   - `internal/provider/mozilla_ai/data/registry.json`
   - `internal/provider/mozilla_ai/data/schema.json`
8. **Commit your changes** with a clear and descriptive message.
9. **Push your branch** to your forked repository.
10. **Open a Pull Request** from your branch to the `main` branch of the upstream `mozilla-ai/mcpd` repository.
    - Follow the [Guidelines for Contributions](#guidelines-for-contributions)
    - Ensure your branch is up-to-date with the main branch before submitting the PR
    - Please follow the PR template, adding as much detail as possible, including how to test the changes
    - Reference the relevant GitHub issue in your PR summary

## Security Vulnerabilities

If you discover a security vulnerability, please **DO NOT** open a public issue. Report it responsibly by following our [Security Policy](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed as described in [LICENSE](LICENSE.md).