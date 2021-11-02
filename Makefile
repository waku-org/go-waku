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

codeclimate-coverage:
	CC_TEST_REPORTER_ID=343d0af350b29aaf08d1e5bb4465d0e21df6298a27240acd2434457a9984c74a
	GIT_COMMIT=$(git log | grep -m1 -oE '[^ ]+$')
	./coverage/cc-test-reporter before-build;\
	go test -count 1 -coverprofile coverage/cover.out ./...;\
	go test -coverprofile coverage/cover.out -json ./... > coverage/coverage.json
    EXIT_CODE=$$?;\
	./coverage/cc-test-reporter after-build -t cover --exit-code $$EXIT_CODE || echo  “Skipping Code Climate coverage upload”

# build a docker image for the fleet
docker-image: DOCKER_IMAGE_TAG ?= latest
docker-image: DOCKER_IMAGE_NAME ?= statusteam/go-waku:$(DOCKER_IMAGE_TAG)
docker-image:
	docker build --tag $(DOCKER_IMAGE_NAME) \
		--build-arg="GIT_COMMIT=$(shell git rev-parse HEAD)" .

build-example-basic2:
	go build examples/basic2/main.go

build-example-chat-2:
	# TODO: require UI + Chat which are gone
	# go build examples/chat2/main.go

build-example-filter2:
	go build examples/filter2/main.go

build-example-peer-events:
	go build examples/peer_events/main.go

build-example: build-example-basic2 build-example-chat-2 build-example-filter2 build-example-peer-events