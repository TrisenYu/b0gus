# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
# Last modified at 2026/04/07 星期二 16:58:38
b0gus_name=b0gus
b0gus_ver=0.0.1
.PHONY: deps debug release test help dry-run perf clean fuzz


### setup local compilers
cc=cc
cxx=g++
clang_check := $(shell clang --version)
clangxx_check := $(shell clang++ --version)
strip_check := $(shell llvm-strip --version)
striper=strip
ifneq ($(clang_check),) # check clang
	cc:=clang
endif
ifneq ($(clangxx_check),) # check clang++
	cxx:=clang++
endif
ifneq ($(strip_check),)
	striper=llvm-strip
endif
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
b0gus_hash=$(shell git describe --long --tags --always --abbrev=40 --dirty || \
echo "uncalculatable")
#### end of setting up local compilers

### set up compiler settings
passing_params=CGO_ENABLED=0 CC=$(cc) CXX=$(cxx)
### set up link flags
# well, $(b0gus_name) has to be precompiled so that the shell pipeline command below can work.
## go tool nm $(b0gus_name) | grep -in "versionStr" | awk '{print $NF}'
must_set_flag=-X 'main.versionStr=$(b0gus_ver)' 		\
              -X 'main.buildTimeStr=$(build_time_str)' 	\
              -X 'main.hashValStr=$(b0gus_hash)'		\
			  -X 'main.builtByStr=$(build_username)'
### end of setting up compiler settings

### entries
# add `-x` flag to audit how go generates b0gus-arm64
# -asan for checking latent memory accessing error
# -race for checking race condition whether exists or not
# go: may not use -race and -asan simultaneously

#### debug for buggy implementation sanitization.
debug_lflags=-gcflags=-l \
			 -ldflags="$(must_set_flag) -X 'b0gus/configs.BuildTypeStr=debug'"
debug: deps
	$(passing_params) go build $(debug_lflags) -o $(b0gus_name)

#### release for production environment.
release_lflags=-gcflags=-l 									 	 \
			   -ldflags="-a -s -w $(must_set_flag) 				 \
			             -X 'b0gus/configs.BuildTypeStr=release' \
						 -compressdwarf=false -buildid="         \
						 -installsuffix cgo 					 \
			   -trimpath -buildmode=exe -pgo off
release: deps
	@$(passing_params) go build $(release_lflags) -o $(b0gus_name)
	$(striper) -s $(b0gus_name)

########### misc opts
# better not execute if there happens to be any error
invoke_protoc=protoc --go_out=. http_cookie.proto; protoc --go_out=. services.proto
deps: clean
	$(passing_params) go mod download
	@-cd databases/rdt-parser && $(gen_script) || echo 								\
	"\033[1;33mantlr4 failed. If having not yet installed antlr4 (version >= 4.13), \
	then it will be better follow the installation tutorial on its official website\033[0m"
	@-cd services && ( $(invoke_protoc) ) || echo 					\
	"\033[1;33mprotoc failed. If having not yet installed protoc, 	\
	command 'sudo apt install protoc-gen-go' is recommended under Debian-based distributions...\033[0m"

dry-run: release
	./$(b0gus_name)
test:
	-cd diff_tests && go test -race
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
	@-rm $(b0gus_name)
perf: # debug
	@echo 'yet to implement'
	@exit 1
# -go tool pprof -http=:8880 assets/cpu.prof
fuzz:
	-cd diff_tests && go test -fuzz=Fuzz
### end of entries
