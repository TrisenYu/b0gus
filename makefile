.PHONY: deps build release


debug: deps
	go build

release:deps
	go build -ldflags="-s -w"

deps:
	go mod download

# TODO: diversity of executable files in different operating systems
run:
	./b0gus