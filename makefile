# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
.PHONY: deps build release

debug: deps
	go build -asan

release:deps
	go build -ldflags="-s -w"

deps:
	go mod download

run:
	./b0gus