.DEFAULT_GOAL := generate

GO ?= go

.PHONY: generate test build

generate:
	GOWORK=off $(GO) generate .

test:
	GOWORK=off $(GO) test -mod=readonly ./...

build:
	GOWORK=off $(GO) build -mod=readonly ./...
