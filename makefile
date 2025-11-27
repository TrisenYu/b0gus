# SPDX-LICENSE-IDENTIFIER: 3-Clause-BSD
.PHONY: deps build release

debug: deps
	go build

release:deps
	go build -ldflags="-s -w"

deps:
	go mod download

run:
	./b0gus