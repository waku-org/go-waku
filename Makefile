
CC_TEST_REPORTER_ID := 343d0af350b29aaf08d1e5bb4465d0e21df6298a27240acd2434457a9984c74a
GO_HTML_COV         := ./coverage.html
GO_TEST_OUTFILE     := ./c.out
CC_PREFIX       	:= github.com/status-im/go-waku

SHELL := bash # the shell used internally by Make

GOBIN ?= $(shell which go)

.PHONY: all build lint test coverage build-example static-library dynamic-library test-c test-c-template mobile-android mobile-ios

ifeq ($(OS),Windows_NT)     # is Windows_NT on XP, 2000, 7, Vista, 10...
 detected_OS := Windows
else
 detected_OS := $(strip $(shell uname))
endif

ifeq ($(detected_OS),Darwin)
 GOBIN_SHARED_LIB_EXT := dylib
  ifeq ("$(shell sysctl -nq hw.optional.arm64)","1")
    # Building on M1 is still not supported, so in the meantime we crosscompile to amd64
    GOBIN_SHARED_LIB_CFLAGS=CGO_ENABLED=1 GOOS=darwin GOARCH=amd64
  endif
else ifeq ($(detected_OS),Windows)
 # on Windows need `--export-all-symbols` flag else expected symbols will not be found in libgowaku.dll
 GOBIN_SHARED_LIB_CGO_LDFLAGS := CGO_LDFLAGS="-Wl,--export-all-symbols"
 GOBIN_SHARED_LIB_EXT := dll
else
 GOBIN_SHARED_LIB_EXT := so
 GOBIN_SHARED_LIB_CGO_LDFLAGS := CGO_LDFLAGS="-Wl,-soname,libgowaku.so.0"
endif

GIT_COMMIT = $(shell git rev-parse --short HEAD)
VERSION = $(shell cat ./VERSION)

BUILD_FLAGS ?= $(shell echo "-ldflags='\
	-X github.com/status-im/go-waku/waku/v2/node.GitCommit=$(GIT_COMMIT) \
	-X github.com/status-im/go-waku/waku/v2/node.Version=$(VERSION)'")


# control rln code compilation
ifeq ($(RLN), true)
BUILD_TAGS := gowaku_rln
endif

all: build

deps: lint-install

build:
	${GOBIN} build -tags="${BUILD_TAGS}" $(BUILD_FLAGS) -o build/waku

chat2:
	pushd ./examples/chat2 && \
	${GOBIN} build -tags="gowaku_rln" -o ../../build/chat2 . && \
	popd

vendor:
	${GOBIN} mod tidy

lint-install:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		bash -s -- -b $(shell ${GOBIN} env GOPATH)/bin v1.50.0

lint:
	@echo "lint"
	@golangci-lint --exclude=SA1019 run ./... --deadline=5m

test:
	${GOBIN} test ./waku/... -coverprofile=${GO_TEST_OUTFILE}.tmp
	cat ${GO_TEST_OUTFILE}.tmp | grep -v ".pb.go" > ${GO_TEST_OUTFILE}
	${GOBIN} tool cover -html=${GO_TEST_OUTFILE} -o ${GO_HTML_COV}

_before-cc:
	CC_TEST_REPORTER_ID=${CC_TEST_REPORTER_ID} ./coverage/cc-test-reporter before-build
	
_after-cc:
	GIT_COMMIT=$(git log | grep -m1 -oE '[^ ]+$') CC_TEST_REPORTER_ID=${CC_TEST_REPORTER_ID} ./coverage/cc-test-reporter after-build --prefix ${CC_PREFIX}

test-ci: _before-cc test _after-cc

generate:
	${GOBIN} generate ./waku/v2/protocol/pb/generate.go
	${GOBIN} generate ./waku/persistence/migrations/sql
	${GOBIN} generate ./waku/v2/protocol/rln/contracts/generate.go
	${GOBIN} generate ./waku/v2/protocol/rln/doc.go
	

coverage:
	${GOBIN} test -count 1 -coverprofile=coverage.out ./...
	${GOBIN} tool cover -html=coverage.out -o=coverage.html

# build a docker image for the fleet
docker-image: DOCKER_IMAGE_TAG ?= latest
docker-image: DOCKER_IMAGE_NAME ?= statusteam/go-waku:$(DOCKER_IMAGE_TAG)
docker-image:
	docker build --tag $(DOCKER_IMAGE_NAME) \
		--build-arg="GIT_COMMIT=$(shell git rev-parse HEAD)" .

build-example-basic2:
	cd examples/basic2 && $(MAKE)

build-example-chat-2:
	cd examples/chat2 && $(MAKE)

build-example-filter2:
	cd examples/filter2 && $(MAKE)

build-example-c-bindings:
	cd examples/c-bindings && $(MAKE)

build-example: build-example-basic2 build-example-chat-2 build-example-filter2 build-example-c-bindings

static-library:
	@echo "Building static library..."
	${GOBIN} build \
		-buildmode=c-archive \
		-tags="${BUILD_TAGS}" \
		-o ./build/lib/libgowaku.a \
		./library/
	@echo "Static library built:"
	@ls -la ./build/lib/libgowaku.*

dynamic-library:
	@echo "Building shared library..."
	$(GOBIN_SHARED_LIB_CFLAGS) $(GOBIN_SHARED_LIB_CGO_LDFLAGS) ${GOBIN} build \
		-buildmode=c-shared \
		-tags="${BUILD_TAGS}" \
		-o ./build/lib/libgowaku.$(GOBIN_SHARED_LIB_EXT) \
		./library/
ifeq ($(detected_OS),Linux)
	cd ./build/lib && \
	ls -lah . && \
	mv ./libgowaku.$(GOBIN_SHARED_LIB_EXT) ./libgowaku.$(GOBIN_SHARED_LIB_EXT).0 && \
	ln -s ./libgowaku.$(GOBIN_SHARED_LIB_EXT).0 ./libgowaku.$(GOBIN_SHARED_LIB_EXT)
endif
	@echo "Shared library built:"
	@ls -la ./build/lib/libgowaku.*

mobile-android:
	gomobile init && \
	${GOBIN} get -d golang.org/x/mobile/cmd/gomobile && \
	gomobile bind -v -target=android -ldflags="-s -w" -tags="${BUILD_TAGS}" $(BUILD_FLAGS) -o ./build/lib/gowaku.aar ./mobile
	@echo "Android library built:"
	@ls -la ./build/lib/*.aar ./build/lib/*.jar

mobile-ios:
	gomobile init && \
	${GOBIN} get -d golang.org/x/mobile/cmd/gomobile && \
	gomobile bind -target=ios -ldflags="-s -w" -tags="nowatchdog ${BUILD_TAGS}" $(BUILD_FLAGS) -o ./build/lib/Gowaku.xcframework ./mobile
	@echo "IOS library built:"
	@ls -la ./build/lib/*.xcframework

install-xtools:
	${GOBIN} install golang.org/x/tools/...@v0.1.10

install-bindata:
	${GOBIN} install github.com/kevinburke/go-bindata/go-bindata@v3.13.0

install-gomobile: install-xtools
	${GOBIN} install golang.org/x/mobile/cmd/gomobile@v0.0.0-20220518205345-8578da9835fd
	${GOBIN} install golang.org/x/mobile/cmd/gobind@v0.0.0-20220518205345-8578da9835fd

build-linux-pkg:
	./scripts/linux/docker-run.sh
	ls -la ./build/*.rpm ./build/*.deb

TEST_MNEMONIC="swim relax risk shy chimney please usual search industry board music segment"

start-ganache:
	docker run -p 8545:8545 --name ganache-cli --rm -d trufflesuite/ganache-cli:latest -m ${TEST_MNEMONIC}

stop-ganache:
	docker stop ganache-cli

test-onchain: BUILD_TAGS += include_onchain_tests
test-onchain:
	${GOBIN} test -v -count 1 -tags="${BUILD_TAGS}" github.com/status-im/go-waku/waku/v2/protocol/rln

