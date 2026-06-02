#!/usr/bin/env bash
# Last modified at 2026/04/05 星期日 02:24:04
set -euo pipefail
shopt -s nullglob

# unix or darwin?
# shellcheck disable=SC2209
grep_check=grep

if [ "$(uname)" = "Darwin" ]; then
    grep_check=ggrep
    # require `brew install gnu-grep`
fi

antlr4 | $grep_check -in "antlr" &> /dev/null
# shellcheck disable=SC2181
if [ "$?" != 0 ]; then
    echo "require ANTLR Parser Generator in your system"
    exit 1
fi

# version number must >= 4.13
to_be_test=$(\
antlr4 2>/dev/null | $grep_check -oiP "version \K[\d\.]+" | \
awk -F'.' '{ major=$1; minor=$2 } major >=4 && minor >= 13')
if [ -z "$to_be_test" ]; then
    echo "require version of antlr >= 4.13"
    exit 1
fi

go version | $grep_check -in "go" &> /dev/null
# shellcheck disable=SC2181
if [ "$?" != 0 ]; then
    echo "require go in your system"
    exit 1
fi

Lang="Go"
g4_files=(*.g4)
# antlr4 defined in /usr/share/bin/: $(which java) -jar antlr4-complete.jar $@
for file in "${g4_files[@]}"; do
	antlr4 -Werror -Dlanguage="$Lang" -no-visitor -listener "$file" -o "$(pwd)/" -Xexact-output-dir
done
if [[ "$Lang" = "Go" ]]; then
	alter_list=$($grep_check "package parser" -rl . | $grep_check -v "gen.*")
	if [ -z "$alter_list" ]; then
		echo ""
	else
		echo "$alter_list" | xargs sed -i "s/package parser/package main/g"
	fi
fi

go generate
rm ./*.interp ./*.tokens
