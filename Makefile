
CC_TEST_REPORTER_ID := 343d0af350b29aaf08d1e5bb4465d0e21df6298a27240acd2434457a9984c74a
GO_HTML_COV         := ./coverage.html
GO_TEST_OUTFILE     := ./c.out
CC_PREFIX       	:= github.com/status-im/go-waku

SHELL := bash # the shell used internally by Make

.PHONY: all build lint test coverage build-example static-library dynamic-library test-c test-c-template

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

BUILD_FLAGS ?= $(shell echo "-ldflags='\
	-X github.com/status-im/go-waku/waku/v2/node.GitCommit=$(GIT_COMMIT)'")

all: build

deps: lint-install

build:
	go build $(BUILD_FLAGS) -o build/waku waku.go

vendor:
	go mod tidy

lint-install:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		bash -s -- -b $(shell go env GOPATH)/bin v1.41.1

lint:
	@echo "lint"
	@golangci-lint --exclude=SA1019 run ./... --deadline=5m

test:
	go test ./waku/... -coverprofile=${GO_TEST_OUTFILE}.tmp
	cat ${GO_TEST_OUTFILE}.tmp | grep -v ".pb.go" > ${GO_TEST_OUTFILE}
	go tool cover -html=${GO_TEST_OUTFILE} -o ${GO_HTML_COV}

_before-cc:
	CC_TEST_REPORTER_ID=${CC_TEST_REPORTER_ID} ./coverage/cc-test-reporter before-build
	
_after-cc:
	CC_TEST_REPORTER_ID=${CC_TEST_REPORTER_ID} ./coverage/cc-test-reporter after-build --prefix ${CC_PREFIX}

test-ci: _before-cc test _after-cc

generate:
	go generate ./waku/v2/protocol/pb/generate.go

coverage:
	go test  -count 1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o=coverage.html

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
	go build \
		-buildmode=c-archive \
		-o ./build/lib/libgowaku.a \
		./library/
	@echo "Static library built:"
	@ls -la ./build/lib/libgowaku.*

dynamic-library:
	@echo "Building shared library..."
	$(GOBIN_SHARED_LIB_CFLAGS) $(GOBIN_SHARED_LIB_CGO_LDFLAGS) go build \
		-buildmode=c-shared \
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
	gomobile bind -target=android -ldflags="-s -w" $(BUILD_FLAGS) -o ./build/lib/gowaku.aar ./mobile
	@echo "Android library built:"
	@ls -la ./build/lib/*.aar ./build/lib/*.jar