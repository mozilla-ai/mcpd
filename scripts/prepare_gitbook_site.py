"""Prepare a deployable GitBook site in site/.

Copies docs/ into site/, adds the GitBook metadata files, fills the CLI
Reference navigation in SUMMARY.md from the generated command pages, and
(when a version is supplied) stamps a version badge on the landing page.

The doc tree is copied verbatim, so relative links between pages keep
resolving without rewriting.

Run the Go documentation generators (make docs-cli docs-api) before this
script so the generated command and API pages exist under docs/.

Usage:
    python scripts/prepare_gitbook_site.py [--version VERSION]
"""

from __future__ import annotations

import argparse
import shutil
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
DOCS_DIR = REPO_ROOT / "docs"
SITE_DIR = REPO_ROOT / "site"
COMMANDS_DIRNAME = "commands"

ROOT_FILES: dict[Path, Path] = {
    REPO_ROOT / ".gitbook.yaml": SITE_DIR / ".gitbook.yaml",
    REPO_ROOT / ".gitbook-branch-readme.md": SITE_DIR / "README.md",
}

IGNORE_PATTERNS = shutil.ignore_patterns(".DS_Store", "__pycache__")

COMMANDS_BEGIN = "<!-- BEGIN COMMANDS -->"
COMMANDS_END = "<!-- END COMMANDS -->"


def command_title(filename: str) -> str:
    """Return the navigation title for a generated command page.

    The root command page (mcpd.md) is titled "Overview".
    Subcommand pages drop the mcpd_ prefix and render underscores as spaces.
    """
    stem = filename[:-3] if filename.endswith(".md") else filename
    if stem == "mcpd":
        return "Overview"
    if stem.startswith("mcpd_"):
        return stem[len("mcpd_") :].replace("_", " ").strip().lower()
    return stem


def command_nav_entries(filenames: list[str]) -> list[str]:
    """Return SUMMARY.md bullet lines for the given command page filenames.

    Overview is listed first, then the remaining commands alphabetically by title.
    """
    titled = [(command_title(name), name) for name in filenames]
    titled.sort(key=lambda entry: (entry[0] != "Overview", entry[0]))
    return [f"* [{title}]({COMMANDS_DIRNAME}/{name})" for title, name in titled]


def render_summary_commands(template: str, entries: list[str]) -> str:
    """Replace the command marker block in a SUMMARY.md body with generated entries."""
    if COMMANDS_BEGIN not in template or COMMANDS_END not in template:
        raise ValueError(f"SUMMARY is missing command markers {COMMANDS_BEGIN} / {COMMANDS_END}")
    begin = template.index(COMMANDS_BEGIN) + len(COMMANDS_BEGIN)
    end = template.index(COMMANDS_END)
    block = "\n" + "\n".join(entries) + "\n" if entries else "\n"
    return template[:begin] + block + template[end:]


def _inject_version_badge(content: str, version: str) -> str:
    """Insert a version indicator after the first top-level heading."""
    lines = content.split("\n")
    in_fence = False
    for i, line in enumerate(lines):
        if line.startswith("```"):
            in_fence = not in_fence
            continue
        if not in_fence and line.startswith("# "):
            j = i + 1
            while j < len(lines) and not lines[j].strip():
                j += 1
            lines[j:j] = [
                '{% hint style="info" icon="tag" %}',
                f"Version: {version}",
                "{% endhint %}",
                "",
            ]
            break
    return "\n".join(lines)


def generate_summary() -> None:
    """Fill the CLI Reference block in site/SUMMARY.md from generated command pages."""
    summary = SITE_DIR / "SUMMARY.md"
    commands_dir = SITE_DIR / COMMANDS_DIRNAME
    filenames = sorted(p.name for p in commands_dir.glob("*.md")) if commands_dir.is_dir() else []
    entries = command_nav_entries(filenames)
    summary.write_text(
        render_summary_commands(summary.read_text(encoding="utf-8"), entries),
        encoding="utf-8",
    )


def stamp_version(version: str) -> None:
    """Stamp the version badge onto the landing page."""
    index = SITE_DIR / "index.md"
    if index.exists():
        index.write_text(
            _inject_version_badge(index.read_text(encoding="utf-8"), version),
            encoding="utf-8",
        )


def parse_args() -> argparse.Namespace:
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--version",
        default="",
        help="Docs version stamped on the landing page (e.g. v1.2.3). Skipped when empty.",
    )
    return parser.parse_args()


def main() -> None:
    """Rebuild the GitBook publication directory from checked-in docs."""
    args = parse_args()

    if SITE_DIR.exists():
        shutil.rmtree(SITE_DIR)

    shutil.copytree(DOCS_DIR, SITE_DIR, ignore=IGNORE_PATTERNS)

    for src, dest in ROOT_FILES.items():
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, dest)

    generate_summary()

    if args.version:
        stamp_version(args.version)

    md_files = sorted(SITE_DIR.rglob("*.md"))
    print(f"Prepared {len(md_files)} markdown files in {SITE_DIR}/")


if __name__ == "__main__":
    main()
