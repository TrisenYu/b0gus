.PHONY: deps build release

deps:
	go mod download

debug: deps
	go build


release:deps
	go build -ldflags="-s -w"

# TODO: diversity of executable files in different operating systems
run:
	./b0gus