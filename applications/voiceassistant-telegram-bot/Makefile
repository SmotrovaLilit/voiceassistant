.PHONY: install-dep lint test

GOLANGCI_LINT_VERSION := v2.15.0

install-dep:
	mkdir -p bin
	curl -sSfL https://golangci-lint.run/install.sh -o bin/install-golangci-lint.sh
	sh bin/install-golangci-lint.sh -b ./bin $(GOLANGCI_LINT_VERSION)
	rm bin/install-golangci-lint.sh

lint:
	./bin/golangci-lint run --fix ./...

test:
	go test ./...
