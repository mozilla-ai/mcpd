"""Tests for prepare_gitbook_site helpers."""

from __future__ import annotations

import pytest

from prepare_gitbook_site import (
    _inject_version_badge,
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


class TestInjectVersionBadge:
    def test_inserts_after_heading_with_blank_line(self) -> None:
        content = "# Title\n\nBody text"
        result = _inject_version_badge(content, "1.2.3")
        expected = (
            '# Title\n\n{% hint style="info" icon="tag" %}\n'
            "Version: 1.2.3\n{% endhint %}\n\nBody text"
        )
        assert result == expected

    def test_no_heading_returns_unchanged(self) -> None:
        content = "No heading here"
        assert _inject_version_badge(content, "1.0.0") == content

    def test_only_first_heading_is_modified(self) -> None:
        content = "# First\n\nText\n\n# Second\n\nMore text"
        result = _inject_version_badge(content, "2.0.0")
        assert result.count("Version: ") == 1

    def test_heading_inside_code_fence_is_skipped(self) -> None:
        content = "```\n# Not a heading\n```\n\n# Real heading\n\nBody"
        result = _inject_version_badge(content, "1.0.0")
        assert "Version: 1.0.0" in result
        assert result.index("Version: 1.0.0") > result.index("# Real heading")

    def test_heading_inside_indented_code_fence_is_skipped(self) -> None:
        content = "    ```\n# Not a heading\n    ```\n\n# Real heading\n\nBody"
        result = _inject_version_badge(content, "1.0.0")
        assert "Version: 1.0.0" in result
        assert result.index("Version: 1.0.0") > result.index("# Real heading")

    def test_includes_mcpd_version_when_provided(self) -> None:
        result = _inject_version_badge("# Title\n\nBody", "v1.2.3", "v0.4.0")
        assert "Version: v1.2.3 (documents mcpd v0.4.0)" in result
