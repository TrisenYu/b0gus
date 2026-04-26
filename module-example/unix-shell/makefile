ifeq ($(OS),Windows_NT)
	$(error program will not properly execute in expectation. Thus abort compilation...)
endif

default:
	go build -ldflags="-X 'b0gus/configs.BuildTypeStr=debug'"
