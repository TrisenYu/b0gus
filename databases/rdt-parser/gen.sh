#!/usr/bin/env bash
set -e
antlr4 | grep -in "antlr" &> /dev/null

# shellcheck disable=SC2181
if [ "$?" != 0 ]; then
    echo "require ANTLR Parser Generator in your system"
    exit 1
fi

Lang="Go"

find *.g4 -print0 | xargs -0 antlr4 -Werror -Dlanguage="$Lang" -no-visitor -listener -o "$(pwd)/" -Xexact-output-dir
if [[ "$Lang" = "Go" ]]; then
    sed -i "s/package parser/package main/g" `grep "package parser" -rl . | grep -v "gen.*"`
fi

$(which go) generate
rm ./*.interp ./*.tokens
