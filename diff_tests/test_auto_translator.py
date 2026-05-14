#!/usr/bin/env python3
# SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
# Last modified at 2026/05/09 星期六 14:54:47
from typing import Optional

import pytest

from auto_translator import markdown_stripper


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
    ("```你好输出\ntoml\n```", 'toml')
])
def test_markdown(a: str, expected: Optional[str]):
    if expected is None:
        assert markdown_stripper(a) is None
    else:
        assert markdown_stripper(a) == expected
