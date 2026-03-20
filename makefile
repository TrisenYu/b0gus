# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
b0gus_name=b0gus
b0gus_ver=0.0.1
.PHONY: deps debug release


### setup local compilers
cc=cc
cxx=g++
clang_path := $(shell command -v clang --version)
clangxx_path := $(shell command -v clang++ --version)
ifneq ($(clang_path),) # check clang
	cc:=clang
endif # clang check
ifneq ($(clangxx_path),) # check clang++
	cxx:=clang++
endif # clang++ check
### end of setting up local compilers


### set up the name of program and build time
ifeq ($(OS),Windows_NT)
define build_time_payload
	powershell -Command "Get-Date -Format 'yyyy-MM-dd HH:mm:ss.fff'"
endef
	b0gus_name+=.exe
else # else linux/darwin
define build_time_payload
	date +"%Y-%m-%d %H:%M:%S.%3N"
endef
endif # operating system check
build_time_str=$(shell command $(build_time_payload))
#### end of setting up local compilers


### set up compiler settings
passing_params=CC=$(cc) CXX=$(cxx) 
### set up link flags
# well, $(b0gus_name) has to be precompiled so that the shell pipeline command below can work.
## go tool nm $(b0gus_name) | grep -in "versionStr" | awk '{print $NF}'
must_set_flag=-X 'main.versionStr=$(b0gus_ver)' -X 'main.buildTimeStr=$(build_time_str)'
link_flags=-ldflags="-s -w $(must_set_flag) -buildid="
### end of setting up compiler settings


### entries
# add `-x` flag to audit how go generates b0gus
# -asan for checking latent memory accessing error
debug: deps
	$(passing_params) go build -o $(b0gus_name)
release:deps
	@$(passing_params) go build $(link_flags) -o $(b0gus_name)
deps:
	-rm $(b0gus_name)
	$(passing_params) go mod download
# $(passing_params) go env > ./.curr_go_env
# TODO: invoke bash files under different directories like generating local trusted certificate as root CA cert
# 		or refresh the definition under databases/rdt-parser/gen.sh or gen.ps1[though yet to implement/complete]

dry-run:
	./$(b0gus_name)
### end of entries