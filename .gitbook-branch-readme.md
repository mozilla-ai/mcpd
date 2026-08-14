# GitBook Documentation Branch

The `gitbook-docs` branch contains **generated** GitBook-compatible documentation,
automatically updated by GitHub Actions when a `docs/v*` release is published.

**Do not edit this branch manually** — all changes will be overwritten.

## How it works

1. Publish a `docs/v*` release (e.g. `docs/v1.0.0`) to trigger a build.
2. The Go generators produce the CLI and OpenAPI pages, and
   `scripts/prepare_gitbook_site.py` assembles them into a site.
3. The contents of `site/` are pushed to this branch.
4. GitBook syncs from this branch.

The workflow can also be triggered manually via `workflow_dispatch`.
