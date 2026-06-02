#!/usr/bin/env python3
# SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
# Last modified at 2026/05/09 星期六 14:54:47
from typing import Any, Optional

import pytest

from tools.auto_translator import markdown_stripper, seize_err_if_any


@pytest.mark.parametrize("a, expected", [
    ("```", None),
    ("``````", None),
    ("`````````", None),
    ("```t```", None),
    ("```\nt\n```", None),
    ("t```", None),
    ("", None),
    ("t", 't'),
    ("```toml\nhello world```", None),
    ("```toml\nhello world\n```", "hello world"),
    ("```tomlhello world", None),
    ("```toml\n你好输出\n```", '你好输出'),
    ("```你好输出\ntoml\n```", 'toml'),
    ("`"*100, None),
    ("```a\nq"*12, None),
    ('{"helo": [1, 2, 3], "world": "456"}', '{"helo": [1, 2, 3], "world": "456"}'),
    ('```json\n{"helo": [1, 2, 3], "world": "456"}\n```', '{"helo": [1, 2, 3], "world": "456"}'),
    ('```json\\n{"helo": [1, 2, 3], "world": "456"}\\n```', None),
    ("a```b", None)
])
def test_markdown(a: str, expected: Optional[str]):
    if expected is None:
        assert markdown_stripper(a) is None
        return
    assert markdown_stripper(a) == expected

@seize_err_if_any()
def _helo():
    return "hello"

@seize_err_if_any()
def _halo():
    return 0

@seize_err_if_any()
def _error():
    raise ValueError("world")

@pytest.mark.parametrize("x, expected", [
    (_helo, "hello"),
    (_halo, 0),
    (_error, None),
])

def test_wrapper(x ,  expected: Any):
    if expected is None:
        assert x() is None
    elif isinstance(expected, int | str):
        assert x() == expected

