"""Tests for check_docs helpers."""

from __future__ import annotations

from check_docs import strip_code_blocks


class TestStripCodeBlocks:
    def test_strips_indented_backtick_fence(self) -> None:
        text = "para\n    ```\n    [link](page.md)\n    ```\nend"
        assert "[link](page.md)" not in strip_code_blocks(text)

    def test_strips_tilde_fence(self) -> None:
        text = "para\n~~~\n[link](page.md)\n~~~\nend"
        assert "[link](page.md)" not in strip_code_blocks(text)

    def test_keeps_prose_links(self) -> None:
        text = "see [link](page.md) for details"
        assert "[link](page.md)" in strip_code_blocks(text)
