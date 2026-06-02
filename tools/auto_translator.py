#!/usr/bin/env python3
# -*- encoding: utf-8 -*-
# SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
# Last modified at 2026/05/28 星期四 17:06:43
"""
[auto_translator] will read configuration from given/default source file
and generate translated files by calling API to interact with LLM.
Only

[TODO]: There might be a better way to update documentation like
    using git diff to compare the modified base-reference and the previous one.
    Thereby avoiding unnecessary token consumption.
"""
import argparse
import http
import json
import re
import sys
import tomllib
from concurrent.futures import ThreadPoolExecutor
from http import HTTPStatus
from pathlib import Path
from traceback import format_exc
from typing import Any, Callable, NoReturn, Optional, Union

import httpx
from loguru import logger
from openai import OpenAI
from pydantic import AliasChoices, AliasGenerator, BaseModel, Field, HttpUrl
from pydantic.alias_generators import to_camel, to_pascal, to_snake

# >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> hard-encoded path string or parameters.
support_lang_arr = [
    "en", "de", "es", "fr",
    "gr", "it", "ja", "ko", "ru",
]
locale_fn = lambda x: str(
    Path(__file__).parent / ".." / "assets" / "locale" / "go-proj" / f"active.{x}.toml"
)
readme_fn = lambda x: str(
    Path(__file__).parent / ".." / "docs" / "readmes" / f"readme-{x}.md"
)


def init_argv() -> argparse.Namespace:
    """
    init_argv defines arguments for further usage.
    :return: parsed argument object.
    """
    _argv_parser = argparse.ArgumentParser(
        description='AI-enhanced translator',
        allow_abbrev=True,
        usage='python auto_translator.py --{with-acceptable-arguments}'
    )
    ## base locale file to other file
    _argv_parser.add_argument(
        '-tn', '--thread-num',
        type=int, default=2,
        help="`thread number` determines how many thread will be put into use"
    )
    _argv_parser.add_argument(
        '-blt', '--base-locale-toml',
        type=str, default=str(
            Path(__file__).parent / ".." / "assets" / "locale" / "go-proj" / "active.zh_cn.toml"
        ),
        help="`base locale toml` points to the absolute path of given toml waiting to be translated"
    )
    _argv_parser.add_argument(
        '-bdm', '--base-document-markdown',
        type=str, default=str(Path(__file__).parent / ".." / "docs" / "help" / "zh_cn.md"),
        help="`markdown reference document` is similar to `base locale toml`"
    )
    _argv_parser.add_argument(
        '-pti', '--path-to-i18n4toml',
        type=str, default=str(Path(__file__).parent / ".." / "assets" / "prompts" / "i18n4toml.txt"),
        help="`path to i18n4toml.txt` is the backup directory "+
             "when there is no `role_prompt` define in llm-conf"
    )
    _argv_parser.add_argument(
        '-lcp', '--llm-conf-path',
        type=str, default=str(Path(__file__).parent / ".." / "configs" / "llm-conf.toml"),
        help="`llm configuration path` is the configuration of this translator and " +
             "stores your secret (e.g. api key, model name) of AI-provider(s)"
    )
    ## Log configuration
    # log path
    _argv_parser.add_argument(
        '-lp', '--log-path',
        type=str, default=str(Path(__file__).parent / ".." / "assets" / "log" / "translator-run.log"),
        help="`log path` will auto-generate log-info during the runtime"
    )
    # log format
    _argv_parser.add_argument(
        '-lf', '--log-format',
        type=str,
        default='<green>{time}</green> [<yellow>{file}:{line}</yellow>' +
                '|<level>{level}</level>] <level>{message}</level>',
        help="`log format` defines the format of log"
    )
    # log level
    _argv_parser.add_argument(
        '-ll', '--log-level',
        type=str, default='DEBUG',
        help="`log level` determines the logging level. " +
             "The options could be {debug, info, warning, error}"
    )
    # log color option
    _argv_parser.add_argument(
        '-elc', '--enable-log-color',
        type=bool, default=True,
        help="when printing to the terminal, `enable log color` will colorize the output of log"
    )
    # log to stdout
    _argv_parser.add_argument(
        "-ds", "--disable-stdout",
        type=bool, default=False,
        help="`disable stdout` will disable the functionality of printing to the "+
             "stdout (i.e. console or terminal)"
    )
    try:
        res = _argv_parser.parse_args()
    except Exception as e:
        print(e)
        sys.exit(1)
    return res

# >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> functionalities

def dual_alias_gen(field_name: str) -> AliasChoices:
    return AliasChoices(
        to_snake(field_name),
        to_camel(field_name),
        to_pascal(field_name)
    )


def seize_err_if_any(logger_enable: bool=True):
    """
    if fn encountered any error, then return None as its result.
    Otherwise, return the expected result(s).
    """
    def dec(fn_with_ret_val):
        def wrapper(*args, **kwargs):
            try:
                return fn_with_ret_val(*args, **kwargs)
            except Exception as e:
                if not logger_enable:
                    return None
                dump_stk = format_exc()
                print(
                    f'an exception was detected: {e}\n' +
                    'current trace stack\n' + dump_stk
                )
            return None
        return wrapper
    return dec


def die_if_err(fn):
    """
    if fn encountered any error,then the whole process
    will terminate as quickly as possible.
    """
    def error_dumper(*args, **kwargs) -> Union[NoReturn, Any]:
        try:
            return fn(*args, **kwargs)
        except Exception as e:
            dump_stk = format_exc()
            print(
                f'an exception was detected: {e}\n' +
                'dumping current trace stack\n' +
                f'{dump_stk}'
            )
            sys.exit(1)
    return error_dumper

def markdown_stripper(s: str) -> Optional[str]:
    res = re.fullmatch(
        # 1. ```-excluded charset
        # 2. start with ```, and concentrate with digit/alphabet characters
        #   2.1 and then a new line
        #   2.2 `-excluded chars
        #   2.3 newline
        #   2.4 ```
        re.compile(r'^(?:[^`]|(?!```).)*$|^```(\w+)\n([^`]*?)\n```$'),
        s
    )
    if res is None:
        return res
    elif res.group(1) is not None:
        # res.group(1) is typically the programming language tag or documentation type.
        # res.group(2) is the content that we want
        return res.group(2)
    elif len(s) > 0:
        # ok, it is a string without ```.
        return s
    return None


class TranslatorConf(BaseModel):
    model_config = {
        "alias_generator": AliasGenerator(validation_alias=dual_alias_gen),
        "extra": "ignore"
    }
    api_key: str = Field(..., min_length=1, strict=True)
    base_url: HttpUrl = Field(..., min_length=1, strict=True)
    model_name: str = Field(..., min_length=1, strict=True)
    role_prompt: str = Field(...)

    balance_api: str # [TODO]: API format checking
    to_be_translated: str
    temperature: float

    @seize_err_if_any()
    def translate_into_lang(
        self, lang_tag: str,
        dst_pattern: Callable[[str], str]=locale_fn
    ) -> None:
        """
        :param lang_tag: the target language tag
        :param dst_pattern:
            since the source reference file has its pattern to the directory,
            the destination file will adapt the pattern and attempt to generate
            the corresponding translated file as well.
        :return: None
        """
        logger.info(lang_tag)
        resp = self.fetch_resp(lang_tag)
        if resp is None:
            logger.warning(f"can't not fetch translation for {lang_tag}")
            return
        logger.info(dst_pattern(lang_tag))
        with open(dst_pattern(lang_tag), "w") as fd:
            fd.write(resp)
        logger.info(f"{lang_tag} done")

    @seize_err_if_any()
    def fetch_resp(self, lang_tag: str) -> Optional[str]:
        client = OpenAI(
            api_key=self.api_key,
            base_url=self.base_url.encoded_string()
        )
        if self.to_be_translated is None or len(self.to_be_translated) <= 0:
            logger.info("do not have text to be translated")
            return None
        # Multiround can be implemented by appending user-messages.
        # in this sense, will require define a member in current pydantic-model.
        # hence leading to the inconvenience in program maintenance and configuration definition.
        completion = client.chat.with_raw_response.completions.create(
            model=self.model_name,
            messages=[
                { "role": "system", "content": self.role_prompt },
                {
                    "role": "user",
                    "content": lang_tag + "\n" + self.to_be_translated
                }
            ],
            stream=False,
        )
        tmp = json.loads(completion.content)
        if completion.status_code != HTTPStatus.OK:
            logger.error(f'{completion.status_code} {completion.content}')
            return None
        return markdown_stripper(tmp["choices"][0]["message"]["content"])

    @seize_err_if_any()
    def reset_content(self, fpath: str) -> None:
        with open(fpath, "r", encoding="utf-8") as fd:
            content = fd.read()
        self.to_be_translated = content


    @seize_err_if_any()
    def balance_checking(self) -> Any:
        if len(self.balance_api) == 0:
            logger.warning("there is not api providing for checking balance, but this function is invoked")
            return None

        query_client = httpx.Client(
            headers={ "Authorization": "Bearer " + self.api_key }
        )
        resp = query_client.get(str(self.base_url) + str(self.balance_api))
        if resp is None:
            return None
        elif resp.status_code != http.HTTPStatus.OK:
            logger.warning(f'{resp.status_code}: {resp.content}')
            return None
        return resp.json()


def init_logger(args: argparse.Namespace) -> None:
    """
    init_logger sets up logger used in current Python script.
    :param args: parsed arguments object.
    :return: no-return.
    """
    logger.remove()
    if not args.disable_stdout:
        logger.add(
            sys.stdout,
            level=args.log_level,
            format=args.log_format,
            colorize=args.enable_log_color,
        )
    logger.add(
        str(Path(args.log_path)),
        level=args.log_level,
        format=args.log_format,
        colorize=False,
        rotation="2MB", compression='zip', encoding='utf-8'
    )

@die_if_err
def init_conf(args: argparse.Namespace) -> TranslatorConf:
    """
    init_conf takes args as its input and return TranslatorConf.
    """
    with open(args.llm_conf_path, "rb") as f:
        conf = tomllib.load(f)
    with open(args.base_locale_toml, "r") as f:
        to_be_translated = f.read()
    conf["to_be_translated"] = to_be_translated
    if "temperature" not in conf:
        conf["temperature"] = 1.0
    if "role_prompt" not in conf or len(conf["role_prompt"]) == 0:
        # load from backup role_prompt file.
        with open(args.path_to_i18n4toml, "r") as f:
            conf["role_prompt"] = f.read()
    return TranslatorConf(**conf)

@die_if_err
def init() -> tuple[TranslatorConf, argparse.Namespace]:
    args = init_argv()
    init_logger(args)
    return init_conf(args), args

def dump_balance(translator: TranslatorConf):
    balance_detail = translator.balance_checking()
    if balance_detail is None:
        logger.warning("can not fetch for balance")
    else:
        logger.info(str(balance_detail))

# >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> entry
if __name__ == "__main__":
    ai_conf, glob_arg = init()
    while True:
        ans = input("Are you sure the reference file is complete? [Y/n]")
        if ans in ("", "y", "yes"):
            break
        if ans in ("n", "no"):
            sys.exit(0)

    with ThreadPoolExecutor(max_workers=glob_arg.thread_num) as executor:
        for task_lang in support_lang_arr:
            executor.submit(ai_conf.translate_into_lang, task_lang)
