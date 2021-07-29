.PHONY: all build lint

all: build

build:
	go build -o build/waku waku.go

lint:
	@echo "lint"
	@golangci-lint --exclude=SA1019 run ./... --deadline=5m