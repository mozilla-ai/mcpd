"""Tests for prepare_gitbook_site helpers."""

from __future__ import annotations

import pytest

from prepare_gitbook_site import (
    _apply_version_badge,
    _render_version_badge,
    command_nav_entries,
    command_title,
    render_summary_commands,
)


class TestCommandTitle:
    def test_root_command_is_overview(self) -> None:
        assert command_title("mcpd.md") == "Overview"

    def test_subcommand_strips_prefix(self) -> None:
        assert command_title("mcpd_daemon.md") == "daemon"

    def test_nested_subcommand_uses_spaces(self) -> None:
        assert command_title("mcpd_config_args.md") == "config args"

    def test_unprefixed_falls_back_to_stem(self) -> None:
        assert command_title("other.md") == "other"


class TestCommandNavEntries:
    def test_overview_first_then_alphabetical(self) -> None:
        files = ["mcpd_daemon.md", "mcpd.md", "mcpd_config_args.md", "mcpd_config.md"]
        assert command_nav_entries(files) == [
            "* [Overview](commands/mcpd.md)",
            "* [config](commands/mcpd_config.md)",
            "* [config args](commands/mcpd_config_args.md)",
            "* [daemon](commands/mcpd_daemon.md)",
        ]

    def test_empty_input_returns_empty(self) -> None:
        assert command_nav_entries([]) == []


class TestRenderSummaryCommands:
    def test_fills_between_markers(self) -> None:
        template = "## CLI Reference\n\n<!-- BEGIN COMMANDS -->\n<!-- END COMMANDS -->\n"
        result = render_summary_commands(template, ["* [Overview](commands/mcpd.md)"])
        assert result == (
            "## CLI Reference\n\n<!-- BEGIN COMMANDS -->\n"
            "* [Overview](commands/mcpd.md)\n<!-- END COMMANDS -->\n"
        )

    def test_replaces_existing_entries_idempotently(self) -> None:
        template = "<!-- BEGIN COMMANDS -->\n* [stale](commands/old.md)\n<!-- END COMMANDS -->\n"
        result = render_summary_commands(template, ["* [Overview](commands/mcpd.md)"])
        assert result == (
            "<!-- BEGIN COMMANDS -->\n* [Overview](commands/mcpd.md)\n<!-- END COMMANDS -->\n"
        )

    def test_missing_markers_raises(self) -> None:
        with pytest.raises(ValueError):
            render_summary_commands("no markers here\n", ["* [x](commands/x.md)"])


class TestRenderVersionBadge:
    def test_leads_with_mcpd_version(self) -> None:
        assert _render_version_badge("v0.1.0", "v0.5.0") == (
            '{% hint style="info" icon="tag" %}\n'
            "mcpd v0.5.0 (docs v0.1.0)\n"
            "{% endhint %}"
        )

    def test_docs_only_when_no_mcpd_version(self) -> None:
        assert _render_version_badge("v0.1.0", "") == (
            '{% hint style="info" icon="tag" %}\n'
            "docs v0.1.0\n"
            "{% endhint %}"
        )


class TestApplyVersionBadge:
    def test_replaces_marker_in_place(self) -> None:
        content = "# mcpd\n\n> tagline\n\n<!-- version-badge -->\n\n---\n"
        assert _apply_version_badge(content, "v0.1.0", "v0.5.0") == (
            "# mcpd\n\n> tagline\n\n"
            '{% hint style="info" icon="tag" %}\n'
            "mcpd v0.5.0 (docs v0.1.0)\n"
            "{% endhint %}\n\n---\n"
        )

    def test_raises_when_marker_missing(self) -> None:
        with pytest.raises(ValueError):
            _apply_version_badge("# mcpd\n\nno marker here\n", "v0.1.0", "v0.5.0")
