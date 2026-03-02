#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
uv sync # create local virtual environment
# For updating dependencies, use following commands.
# uv pip freeze > requirements.txt
# uv add -r requirements.txt
"""
from pathlib import Path 
import os

from dotenv import load_dotenv

CURR_DIR = Path(__file__).parent
_ai_translator_conf_path = CURR_DIR/".."/"configs"/".translator_api" 
load_dotenv(dotenv_path=_ai_translator_conf_path.absolute(), verbose=True)

if __name__ == "__main__":
    # gain api from this configuration.
    translator_password = os.getenv("password")
    translator_token = os.getenv("token")
    translator_url = os.getenv("url")
    translator_prompt = os.getenv("prompt")
    # so which AI api do we import?
    # What about Genimi, ChatGPT or Doubao, DeepSeek as options?
    # TODO: send to AI translator for automatically creating translation
    ref_toml_path = CURR_DIR/"locale"/"active.en.toml"
    payload = ref_toml_path.read_text(encoding="utf-8")
