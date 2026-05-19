# SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
# Last modified at 2026/05/19 星期二 17:02:58
b0gus_name=b0gus
b0gus_ver=0.1.2

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
	# compiler toolchain
	clang_check := $(shell where clang >nul 2>&1)
	clangxx_check := $(shell where clang++ >nul 2>&1)
	strip_check := $(shell where llvm-strip >nul 2>&1)
	# docker
	dock_check := $(shell where docker >nul 2>&1)
	nerd_check := $(shell where nerdctl --version >nul 2>&1)
	# python
	uv_check := $(shell where uv >nul 2>&1)
	pyenv_check := $(shell dir .venv 2>nul)
	activate_pyenv := .venv/Lib/activate
else # linux/darwin
define build_time_payload
	command date +"%Y-%m-%d %H:%M:%S.%3N"
endef
	build_username:=$(shell echo $$USER)
	gen_script:=bash gen.sh
	# compiler toolchain
	clang_check := $(shell command -v clang &>/dev/null)
	clangxx_check := $(shell command -v clang++ &>/dev/null)
	strip_check := $(shell command -v llvm-strip &>/dev/null)
	# docker
	dock_check := $(shell command -v docker &>/dev/null)
	nerd_check := $(shell command -v nerdctl --version &>/dev/null)
	# python
	uv_check := $(shell command -v uv --version &>/dev/null)
	pyenv_check := $(shell ls .venv 2>/dev/null)
	activate_pyenv := . .venv/bin/activate
endif # OS check


#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- compiler toolchain check
ifneq ($(clang_check),) # check clang
	cc:=clang
endif
ifneq ($(clangxx_check),) # check clang++
	cxx:=clang++
endif
ifneq ($(strip_check),) # check llvm-strip
	striper=llvm-strip
endif

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- docker-cli check
ifeq ($(strip $(dock_check)),) # first nested check for docker-cli
ifeq ($(strip $(nerd_check)),) # second nested check for available docker-cli
$(warning can not build by docker since docker has not be installed.)
else
	dock=nerdctl
endif # nerd_check
else
	dock=docker
endif # dock_check

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- set up compiler settings
build_time_str=$(shell $(build_time_payload))
# docker-built will use uncalculatable because we exclude the .git directory in .dockerignore
b0gus_hash=$(shell git describe --long --tags --always --abbrev=40 --dirty || echo "uncalculatable")

passing_params=CC=$(cc) CXX=$(cxx) CGO_ENABLED=0
ifneq ($(osType),)
	passing_params+="GOOS=$(osType)"
endif
ifneq ($(Arch),)
	passing_params+="GOARCH=$(Arch)"
endif

### set up link flags
# well, $(b0gus_name) has to be precompiled so that the shell pipeline command below can work.
## go tool nm $(b0gus_name) | grep -in "versionStr" | awk '{print $NF}'
must_set_flag=-X 'main.versionStr=$(b0gus_ver)' 		\
              -X 'main.buildTimeStr=$(build_time_str)' 	\
              -X 'main.hashValStr=$(b0gus_hash)'		\
			  -X 'main.builtByStr=$(build_username)'

help:
	@echo "[makefile] usages:\n"                                        \
	"    help      - (default) print this help\n\n"                     \
	"    debug     - compile b0gus for debugging\n" 		            \
	"    release   - compile b0gus for releasing\n\n" 		            \
	"    mock      - generate internal/mock structure for diff_tests\n" \
	"    test      - test testcases under ./diff_tests/\n" 	            \
	"    dry-run   - trial\n" 								            \
	"    perf      - evaluate performance\n"                            \
	"    clean     - clean build files and registered files\n"          \
	"    fuzz      - use {go fuzz} to fuzz available testcases\n\n"     \
	"docker-build  - build b0gus by docker\n\n"                         \
	"uv-fresh-dep  - update the dependencies in requirements.txt\n"     \
	"    pylint    - lint for python scripts or codes\n"                \
	"    pytest    - test for python scripts or codes"
phony += help


#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- debug setting for address/thread sanitizer.
# add `-x` flag to audit how go generates b0gus
# -asan for checking latent memory accessing error
# -race for checking race condition whether exists or not
# 	go: may not use -race and -asan simultaneously
debug_link_opts=-gcflags="-l -m" \
			 -ldflags="$(must_set_flag) -X 'b0gus/configs.BuildTypeStr=debug'"
debug: deps
	-$(passing_params) go build $(debug_link_opts) -o $(b0gus_name)
phony += debug

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- release setting for production environment.
# influenced by .gopclntab
release_link_opts=-trimpath -pgo=off            \
	-buildvcs=false -buildmode=pie              \
	-gcflags="-l -linkshared -smallframes"      \
	-ldflags="-s -w $(must_set_flag)            \
        -X 'b0gus/configs.BuildTypeStr=release' \
        -buildid= "


release: deps
	-$(passing_params) go build $(release_link_opts) -o $(b0gus_name)
# yes, strip the symbols
	-$(striper) --strip-all $(b0gus_name)
phony += release

# better not execute if there happens to be any error
invoke_protoc=protoc --go_out=. http_aux.proto; \
			  protoc --go_out=. services.proto
deps: clean
	@go mod tidy
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
	-cd diff_tests && go test -race -v
phony += test


clean:
	@-rm $(b0gus_name)
phony += clean

perf: # debug
# -go tool pprof -http=:8880 assets/cpu.prof
	@$(error yet to implement)
phony += perf

fuzz: test
# fuzz in 2 minutes
# that is weird... we can not run go fuzzing test sequentially
	cd diff_tests &&                                                  \
	go test -v -fuzz=FuzzShell -fuzztime=120s -parallel=2 -run=^$$ && \
	go test -v -fuzz=FuzzHash -fuzztime=120s -parallel=4 -run=^$$
phony += fuzz

mock:
# TODO: correct the alias for mockgen
	~/go/bin/mockgen                                \
		-source=services/abstract_tcpip.go          \
		-destination=internal/mock/mock_net_serv.go \
		-package=mock
	~/go/bin/mockgen                          \
		-source=databases/db_aux.go           \
		-destination=internal/mock/mock_db.go \
		-package=mock

phony += mock

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- docker related

docker-build:
	@$(dock) run --rm --entrypoint /bin/cat b0gus-img /app/b0gus > $(b0gus_name)
phony += docker-build

__docker-comp:
# [TODO]: test the correctness and the effectiveness of current `docker-compose.yaml`.
	@$(dock) compose -d --build -f ./docker-compose.yaml up
phony += __docker-comp

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- python related
__uv-check:
ifeq ($(strip $(uv_check)),)
	$(error require uv to manage your local environment for python)
endif
phony += __uv-check

__pyenv-check: __uv-check
ifeq ($(strip $(pyenv_check)),)
	@uv venv && uv sync && $(activate_pyenv)
else
	@$(activate_pyenv)
endif
phony += __pyenv-check

uv-fresh-dep: __pyenv-check
# other useful command:
# 	@uv sync --upgrade # upgrade the dependencies
	@uv pip freeze > requirements.txt
	@uv add -r requirements.txt
phony += uv-fresh-dep

pylint: __pyenv-check
# check and try the basic fix by ruff
	@$(activate_pyenv) && ruff check --fix
phony += pylint

pytest: __pyenv-check
	@uv run pytest
phony += pytest

.PHONY: $(phony)
