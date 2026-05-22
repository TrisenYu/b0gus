ifeq ($(OS),Windows_NT)
	$(error program will not execute correctly in expectation. Thus abort current compilation...)
endif

default:
	go build -ldflags="-X 'b0gus/configs.BuildTypeStr=debug'"
