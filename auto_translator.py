#!/usr/bin/env python3
# -*- encoding: utf-8 -*-
# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
import argparse
import sys
import tomllib
import json
# import httpx

from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from traceback import format_exc
from typing import Optional, Callable

from loguru import logger
from openai import OpenAI
from pydantic import BaseModel, Field, HttpUrl, AliasGenerator, AliasChoices
from pydantic.alias_generators import to_snake, to_camel, to_pascal

# >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> hard-encoded path string or parameters.
support_lang_arr = [
	"en", "de", "en", "es", "fr",
	"gr", "it", "ja", "ko", "ru",
]
locale_fn = lambda x: str(
	Path(__file__).parent / "assets" / "locale" / f"active.{x}.toml"
)
readme_fn = lambda x: str(
	Path(__file__).parent / "docs" / "readmes" / f"readme-{x}.md"
)

def init_argv() -> argparse.Namespace:
	"""
	init_argv defines arguments for further usage.
	:return: parsed argument object.
	"""
	_argv_parser = argparse.ArgumentParser(
		description='AI-enhanced translator',
		allow_abbrev=True, usage=''
	)
	## base locale file to other file
	_argv_parser.add_argument(
		'-tn', '--thread-num',
		type=int, default=2,
	)
	_argv_parser.add_argument(
		'-blf', '--base-locale-file',
		type=str, default=str(Path(__file__).parent / "assets" / "locale" / "active.zh_cn.toml")
	)
	_argv_parser.add_argument(
		'-tcp', '--translator-conf-path',
		type=str, default=str(Path(__file__).parent /  "configs" / "translator-conf.toml")
	)
	_argv_parser.add_argument(
		'-mrd', '--markdown-ref-document',
		type=str, default=str(Path(__file__).parent / "docs" / "help" / "readme-zh_cn.md")
	)
	## Log configuration
	# log path
	_argv_parser.add_argument(
		'-lp', '--log-path',
		type=str, default=str(Path(__file__).parent / "assets" / "log" / "translator-run.log")
	)
	# log format
	_argv_parser.add_argument(
		'-lf', '--log-format',
		type=str,
		default='<green>{time}</green> [<yellow>{file}:{line}</yellow>' +
		        '|<level>{level}</level>] <level>{message}</level>'
	)
	# log level
	_argv_parser.add_argument(
		'-ll', '--log-level',
		type=str, default='DEBUG'
	)
	# log color option
	_argv_parser.add_argument(
		'-elc', '--enable-log-color',
		type=bool, default=True,
	)
	# log to stdout
	_argv_parser.add_argument(
		"-ds", "--disable-stdout",
		type=bool, default=False,
	)
	try:
		res = _argv_parser.parse_args()
	except Exception as e:
		print(e)
		exit(1)
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
	if fn encountered any error, then return None as the result.
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
				logger.error(
					f'an exception was detected: {e}\n'
					f'current trace stack\n'
					f'{dump_stk}'
				)
			return None
		return wrapper
	return dec


def die_if_err(fn):
	"""
	if fn encountered any error, then the whole process will terminate as quickly as possible.
	"""
	def error_dumper(*args, **kwargs):
		try:
			return fn(*args, **kwargs)
		except Exception as e:
			dump_stk = format_exc()
			print(
				f'an exception was detected: {e}\n'
				f'dumping current trace stack\n'
				f'{dump_stk}'
			)
			exit(1)
	return error_dumper


class TranslatorConf(BaseModel):
	model_config = {
		"alias_generator": AliasGenerator(validation_alias=dual_alias_gen),
		"extra": "ignore"
	}
	api_key: str = Field(..., min_length=1, strict=True)
	base_url: HttpUrl = Field(..., min_length=1, strict=True)
	model_name: str = Field(..., min_length=1, strict=True)
	role_prompt: str = Field(...)
	temperature: float
	to_be_translated: str

	@seize_err_if_any()
	def translate_into_lang(
		self, lang_tag: str,
		dst_pattern: Callable[[str], str]=locale_fn
	) -> None:
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
		completion = client.chat.with_raw_response.completions.create(
			model=self.model_name,
			messages=[
				{"role": "system", "content": self.role_prompt },
				{
					"role": "user",
					"content": lang_tag + "\n" + self.to_be_translated
				}
			],
			stream=False,
			# timeout=httpx.Timeout(120, connect=10),
		)
		tmp = json.loads(completion.content)
		if completion.status_code != 200:
			logger.error(f'{completion.status_code} {completion.content}')
			return None
		return tmp["choices"][0]["message"]["content"]

	@seize_err_if_any()
	def reset_content(self, fpath: str) -> None:
		with open(fpath, "r", encoding="utf-8") as fd:
			content = fd.read()
		self.to_be_translated = content

	def expense_checking(self):
		...


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
	with open(args.translator_conf_path, "rb") as f:
		conf = tomllib.load(f)
	with open(args.base_locale_file, "r") as f:
		to_be_translated = f.read()
	conf["to_be_translated"] = to_be_translated
	if "temperature" not in conf:
		conf["temperature"] = 1.0
	return TranslatorConf(**conf)

@die_if_err
def init() -> tuple[TranslatorConf, argparse.Namespace]:
	args = init_argv()
	init_logger(args)
	return init_conf(args), args

# >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> entry

if __name__ == "__main__":
	ai_conf, glob_arg = init()
	while True:
		ans = input("Are you sure the reference file is complete? [Y/n]")
		if ans in ("", "y", "yes"):
			break
		if ans in ("n", "no"):
			exit(0)
	with ThreadPoolExecutor(max_workers=glob_arg.thread_num) as executor:
		for task_lang in support_lang_arr:
			executor.submit(ai_conf.translate_into_lang, task_lang)
