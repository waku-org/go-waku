.PHONY: all build lint test coverage build-example

all: build

deps: lint-install

build:
	go build -o build/waku waku.go

vendor:
	go mod tidy

lint-install:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		bash -s -- -b $(shell go env GOPATH)/bin v1.41.1

lint:
	@echo "lint"
	@golangci-lint --exclude=SA1019 run ./... --deadline=5m

test:
	go test -v -failfast ./...

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

build-example: build-example-basic2 build-example-chat-2 build-example-filter2