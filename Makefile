.PHONY: all build

all: build

build:
	go build -o build/waku waku.go
