# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
# Last modified at 2026/04/22 星期三 13:06:19
b0gus_name=b0gus
b0gus_ver=0.1.1

phony=
Arch=
osType=

cc=gcc
cxx=g++
striper=strip
dock=docker

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- set up the name of program and build time
ifeq ($(OS),Windows_NT)
define build_time_payload
	powershell -Command "Get-Date -Format 'yyyy-MM-dd HH:mm:ss.fff'"
endef
	b0gus_name+=.exe
	build_username:=$(shell cmd /c "echo %USERNAME%")
	gen_script:= powershell gen.ps1
	dock_check := $(shell where docker >nul 2>&1)
	nerd_check := $(shell where nerdctl --version >nul 2>&1)
	clang_check := $(shell where clang >nul 2>&1)
	clangxx_check := $(shell where clang++ >nul 2>&1)
	strip_check := $(shell where llvm-strip >nul 2>&1)
else # linux/darwin
define build_time_payload
	command date +"%Y-%m-%d %H:%M:%S.%3N"
endef
	build_username:=$(shell echo $$USER)
	gen_script:=bash gen.sh
	dock_check := $(shell command -v docker &>/dev/null)
	nerd_check := $(shell command -v nerdctl --version &>/dev/null)
	clang_check := $(shell command -v clang &>/dev/null)
	clangxx_check := $(shell command -v clang++ &>/dev/null)
	strip_check := $(shell command -v llvm-strip &>/dev/null)
endif # OS check


#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- try to change compiler
ifneq ($(clang_check),) # check clang
	cc:=clang
endif
ifneq ($(clangxx_check),) # check clang++
	cxx:=clang++
endif
ifneq ($(strip_check),) # check llvm-strip
	striper=llvm-strip
endif

ifeq ($(strip $(dock_check)),)
ifeq ($(strip $(nerd_check)),)
	$(warning can not build by docker since docker has not be installed.)
else
	dock=nerdctl
endif # nerd_check
else
	dock=docker
endif # dock_check

build_time_str=$(shell $(build_time_payload))
# docker-built will be uncalculatable because we exclude the .git directory in .dockerignore
b0gus_hash=$(shell git describe --long --tags --always --abbrev=40 --dirty || echo "uncalculatable")


#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- set up compiler settings
passing_params=CGO_ENABLED=0 CC=$(cc) CXX=$(cxx)
ifneq ($(osType),)
	passing_params+="GOOS=$(osType)"
endif
ifneq ($(Arch),)
	passing_params+="GOARCH=$(Arch)"
endif

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- set up link flags
# well, $(b0gus_name) has to be precompiled so that the shell pipeline command below can work.
## go tool nm $(b0gus_name) | grep -in "versionStr" | awk '{print $NF}'
must_set_flag=-X 'main.versionStr=$(b0gus_ver)' 		\
              -X 'main.buildTimeStr=$(build_time_str)' 	\
              -X 'main.hashValStr=$(b0gus_hash)'		\
			  -X 'main.builtByStr=$(build_username)'


#### entries
# add `-x` flag to audit how go generates b0gus
# -asan for checking latent memory accessing error
# -race for checking race condition whether exists or not
# 	go: may not use -race and -asan simultaneously

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- debug setting for buggy implementation sanitization.
debug_lflags=-gcflags="-l -m" \
			 -ldflags="$(must_set_flag) -X 'b0gus/configs.BuildTypeStr=debug'"
debug: deps
	$(passing_params) go build $(debug_lflags) -o $(b0gus_name)
phony += debug


#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- release setting for production environment.
release_lflags=-gcflags=-l 									 	 \
			   -ldflags="-a -s -w $(must_set_flag) 				 \
			             -X 'b0gus/configs.BuildTypeStr=release' \
						 -compressdwarf=false -buildid="         \
						 -installsuffix cgo 					 \
			   -trimpath -buildmode=exe -pgo off
release: deps
	@$(passing_params) go build $(release_lflags) -o $(b0gus_name)
	$(striper) -s $(b0gus_name)
phony += release


#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- misc opts
# better not execute if there happens to be any error
invoke_protoc=protoc --go_out=. http_aux.proto; \
			  protoc --go_out=. services.proto
deps: clean
	$(passing_params) go mod download
	@-cd databases/rdt-parser && $(gen_script) || echo 			  \
	"\033[1;33mantlr4 failed. If having not yet installed antlr4 (version >= 4.13), \
	then it will be better follow the installation tutorial on its official website\033[0m"
	@-cd services && ( $(invoke_protoc) ) || echo 				  \
	"\033[1;33mprotoc failed. If having not yet installed protoc, \
	command 'sudo apt install protoc-gen-go' is recommended in Debian-based distributions...\033[0m"
phony += deps

dry-run: release
	./$(b0gus_name)
phony += dry-run

test:
	-cd diff_tests && go test -race
phony += test

help:
	@echo "makefile usages:\n" 								   \
	"    help     -- print this help\n" 					   \
	"    debug    -- compile b0gus for debugging\n" 		   \
	"    release  -- compile b0gus for releasing\n" 		   \
	"    test     -- test testcases under ./diff_tests/\n" 	   \
	"    dry-run  -- trial\n" 								   \
	"    perf     -- performance evaluation\n" 				   \
	"    clean    -- clean build files and registered files\n" \
	"    fuzz     -- go fuzz available testcases\n"            \
	"docker-build -- build b0gus by docker\n"                  \
	"uv-dep-fresh -- modify dependencies in requirements.txt"
phony += help

clean:
	@-rm $(b0gus_name)
phony += clean

perf: # debug
# -go tool pprof -http=:8880 assets/cpu.prof
	@echo 'yet to implement'
	@exit 1
phony += perf

fuzz:
# fuzz in 2 minutes
# that is weird... we can not run go fuzzing test in sequence
	cd diff_tests && go test -v -fuzz=FuzzShell -fuzztime=120s -parallel=2 -run=^$$ && \
	go test -v -fuzz=FuzzHash -fuzztime=120s -parallel=4 -run=^$$
phony += fuzz

docker-build:
	@$(dock) build -t b0gus-img -f buildImg.Dockerfile .
	@$(dock) run --rm --entrypoint /bin/cat b0gus-img /app/b0gus > $(b0gus_name)
# other cli:
# 	nerdctl build -t b0gus-image .
# 	nerdctl run --rm --entrypoint /bin/cat b0gus-image /app/b0gus > ./b0gus-exe
phony += docker-build

uv-dep-fresh:
# this option is not strictly necessary...
# must `source .venv/bin/activate` or `.venv\bin\activate` first.
	@uv pip freeze > requirements.txt
	@uv add -r requirements.txt
phony += uv-dep-fresh

.PHONY: $(phony)
