# SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
# Last modified at 2026/06/02 星期二 14:24:45
b0gus_name = b0gus
# milestone.major.minor, no patch at present
b0gus_ver = 0.3.0
Arch =
osType =

phony =

cc = gcc
cxx = g++
striper = strip
dock = docker
go_install_path := $(shell go env GOPATH)/bin/
#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- set up the name of program and build time
ifeq ($(OS),Windows_NT)
define build_time_payload
	powershell -Command "Get-Date -Format 'yyyy-MM-dd HH:mm:ss.fff'"
endef
	b0gus_name+=.exe
	build_username := $(shell cmd /c "echo %USERNAME%")
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
	build_username := $(shell echo $$USER)
	gen_script := bash gen.sh
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
build_time_str = $(shell $(build_time_payload))
# docker-built will use uncalculatable because we exclude the .git directory in .dockerignore
b0gus_hash = $(shell git describe --long --tags --always --abbrev=40 --dirty || echo "uncalculatable")

passing_params = CC=$(cc) CXX=$(cxx) CGO_ENABLED=0
ifneq ($(osType),)
	passing_params+="GOOS=$(osType)"
endif
ifneq ($(Arch),)
	passing_params+="GOARCH=$(Arch)"
endif

# help first.
help:
# bake-format off
	@echo "[makefile] usages:\n"                                            \
	"    help         - (default) print this help\n\n"                      \
	"    debug        - compile b0gus for debugging\n"                      \
	"    release      - compile b0gus for releasing\n\n"                    \
	"    mock         - generate internal/mock structure for diff_tests/\n" \
	"    test         - test testcases under diff_tests/\n"                 \
	"    dry-run      - trial\n"                                            \
	"    perf         - evaluate performance\n"                             \
	"    clean        - clean build files and registered files\n"           \
	"    fuzz         - use {go fuzz} to fuzz available testcases\n"        \
	"    golint       - use golint-cli to lint current codes\n\n"           \
	"    docker-build - build b0gus by docker\n\n"                          \
	"    uv-fresh-dep - update the dependencies in requirements.txt\n"      \
	"    pylint       - lint for python scripts\n"                          \
	"    pytest       - test for python scripts\n\n"                        \
	"    test-all     - test all testcases\n"                               \
	"    lint-all     - lint all available codes\n\n"                       \
	"optional parameters:\n"                                                \
	"    Arch         - Target ISA architecture\n"                          \
	"    osType       - Target OS binary format"
# bake-format on

phony += help

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- debug setting for address/thread sanitizer.
# add `-x` flag to audit how go generates b0gus
# -asan for checking latent memory accessing error
# -race for checking race condition whether exists or not
# 	go: may not use -race and -asan simultaneously

### set up link flags
# well, $(b0gus_name) has to be precompiled so that the shell pipeline command below can work.
## go tool nm $(b0gus_name) | grep -in "versionStr" | awk '{print $NF}'
# bake-format off
must_set_flag = -X 'b0gus/configs.versionStr=$(b0gus_ver)'      \
              -X 'b0gus/configs.buildTimeStr=$(build_time_str)' \
              -X 'b0gus/configs.hashValStr=$(b0gus_hash)'       \
              -X 'b0gus/configs.builtByStr=$(build_username)'
# bake-format on

debug_link_opts = -gcflags="-l -m" \
			 -ldflags="$(must_set_flag) -X 'b0gus/configs.BuildTypeStr=debug'"
debug: deps
	-$(passing_params) go build $(debug_link_opts) -o $(b0gus_name)
phony += debug

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- release setting for production environment.
# bake-format off
release_link_opts = -trimpath -pgo=off      \
	-buildvcs=false -buildmode=pie          \
	-gcflags="-l -linkshared -smallframes"  \
	-ldflags="-s -w $(must_set_flag)        \
	-X 'b0gus/configs.BuildTypeStr=release' \
	-buildid= "
# bake-format on

release: deps
	-$(passing_params) go build $(release_link_opts) -o $(b0gus_name)
# yes, strip the symbols
	-$(striper) --strip-all $(b0gus_name)
	-$(striper) -R .go.buildinfo $(b0gus_name)
phony += release

# better not execute if there happens to be any error
invoke_protoc = protoc --go_out=. http_aux.proto; \
				protoc --go_out=. --go-grpc_out=. services.proto;
deps: clean
	@go mod tidy
	$(passing_params) go mod download
	@-cd tools/rdt-parser && $(gen_script) || echo \
	"\033[1;33m'antlr4' failed. If having not yet installed antlr4 (version >= 4.13), \
	then it will be better follow the installation tutorial on its official website\033[0m"
	@-cd services && ( $(invoke_protoc) ) || echo  \
	"\033[1;33m'protoc' or 'protoc-gen-grpc' might failed. If having not yet installed protoc or \
	protoc-gen-go-grpc, command 'sudo apt install protoc-gen-go protoc-gen-go-grpc' is recommended \
	in Debian-based distributions...\033[0m"
phony += deps

dry-run: release
	./$(b0gus_name)
phony += dry-run

test:
# bake-format off
	go test -timeout=300s -race -v \
		-failfast -count=1         \
		-covermode=atomic          \
		-coverprofile=coverage.out \
		-coverpkg=./... ./... &&   \
		go tool cover -func=coverage.out | grep -iI "total"
# bake-format on
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
# [TODO]: that is weird... we can not run go fuzzing test sequentially
# bake-format off
	cd diff_tests &&                                                  \
	go test -v -fuzz=FuzzShell -fuzztime=120s -parallel=2 -run=^$$ && \
	go test -v -fuzz=FuzzHash -fuzztime=120s -parallel=4 -run=^$$
# bake-format on
phony += fuzz

mock:
# bake-format off
	-$(go_install_path)mockgen                      \
		-source=net_aux/abstract_tcpip.go           \
		-destination=internal/mock/mock_net_serv.go \
		-package=mock
	-$(go_install_path)mockgen                \
		-source=databases/db_aux.go           \
		-destination=internal/mock/mock_db.go \
		-package=mock
# bake-format on
phony += mock

golint:
	-$(go_install_path)golangci-lint run ./...
phony += golint

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- docker related
docker-build:
	@$(dock) build -t b0gus-img -f buildImg.Dockerfile .
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
# if this command failed, reinstall uv instead
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

#-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=- integrated formating or testing
test-all: pytest fuzz
phony += test-all

lint-all: pylint golint
phony += lint-all

.PHONY: $(phony)
