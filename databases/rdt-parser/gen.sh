#!/usr/bin/env bash
set -e
antlr4 | grep "ANTLR Parser Generator" &> /dev/null

if [ "$?" != 0 ]; then
    echo "require ANTLR Parser Generator in your system"
    exit 1
fi

Lang="Go"

ls *.g4 | xargs antlr4 -Werror -Dlanguage="$Lang" -no-visitor -listener -o "$(pwd)/" -Xexact-output-dir
if [[ "$Lang" -eq "Go" ]]; then
    sed -i "s/package parser/package main/g" `grep "package parser" -rl . | grep -v "gen.*"` 
fi

$(which go) generate
rm *.interp *.tokens
