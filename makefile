# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
.PHONY: deps build release

cc=cc
cxx=g++
b0gus_name=b0gus

### setup local compilers
clang_path := $(shell command -v clang --version)
clangxx_path := $(shell command -v clang++ --version)
ifneq ($(clang_path),)
	cc:=clang
endif # clang check
ifneq ($(clangxx_path),)
	cxx:=clang++
endif # clang++ check

ifeq ($(OS),Windows_NT)
    b0gus_name+=.exe
endif # operating system check
#### end of setup local compilers

passing_params=CC=$(cc) CXX=$(cxx) 

### entries
# add `-x` flag to audit how go generates b0gus
debug: deps
	$(passing_params) go build -asan -o $(b0gus_name)

release:deps
	$(passing_params) go build -ldflags="-s -w -buildid=" -o $(b0gus_name)

deps:
	$(passing_params) go mod download
	$(passing_params) go env > ./.curr_go_env

run:
	./$(b0gus_name)
#### end of entries