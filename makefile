# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
# Last modified at 2026/03/22 星期日 15:40:08
b0gus_name=b0gus
b0gus_ver=0.0.1
.PHONY: deps debug release test help dry-run perf clean


### setup local compilers
cc=cc
cxx=g++
clang_path := $(shell clang --version)
clangxx_path := $(shell clang++ --version)
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
	build_username:=$(shell cmd /c "echo %USERNAME%")
	gen_script:= powershell gen.ps1
else # else linux/darwin
define build_time_payload
	command date +"%Y-%m-%d %H:%M:%S.%3N"
endef
	build_username:=$(shell echo $$USER)
	gen_script:= bash gen.sh
endif # operating system check
build_time_str=$(shell $(build_time_payload))
b0gus_hash=$(shell git describe --long --tags --always --abbrev=40 --dirty || echo "uncalculatable")
#### end of setting up local compilers

### set up compiler settings
passing_params=CC=$(cc) CXX=$(cxx)
### set up link flags
# well, $(b0gus_name) has to be precompiled so that the shell pipeline command below can work.
## go tool nm $(b0gus_name) | grep -in "versionStr" | awk '{print $NF}'
must_set_flag=-X 'main.versionStr=$(b0gus_ver)' 		\
              -X 'main.buildTimeStr=$(build_time_str)' 	\
              -X 'main.hashValStr=$(b0gus_hash)'		\
			  -X 'main.builtByStr=$(build_username)'
### end of setting up compiler settings


### entries
# add `-x` flag to audit how go generates b0gus
# -asan for checking latent memory accessing error
# -race for checking race condition whether exists or not
# go: may not use -race and -asan simultaneously
#### debug for buggy implementation sanitization.
debug_lflags=-ldflags="$(must_set_flag) -X 'b0gus/configs.BuildTypeStr=debug'"
debug: deps
	$(passing_params) go build $(debug_lflags) -o $(b0gus_name)

#### release for production environment.
release_lflags=-ldflags="-s -w $(must_set_flag) -X 'b0gus/configs.BuildTypeStr=release' -buildid="
release: deps
	@$(passing_params) go build $(release_lflags) -o $(b0gus_name)

###########
deps: clean
	$(passing_params) go mod download
	cd databases/rdt-parser && $(gen_script)
dry-run:
	./$(b0gus_name)
test:
	@cd diff_tests
	@go test
help:
	@echo "makefile usages:\n" 								\
	"    help     -- print help\n" 							\
	"    debug    -- compile b0gus for debugging\n" 		\
	"    release  -- compile b0gus for releasing\n" 		\
	"    test     -- test testcases under ./diff_tests/\n" 	\
	"    dry-run  -- trial\n" 								\
	"    perf     -- performance evaluation\n" 				\
	"    clean    -- clean build files and registered files"
clean:
	-rm $(b0gus_name)
perf:
	@echo "yet to implement"
	@exit 1
### end of entries
