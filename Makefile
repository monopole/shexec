GOBIN = $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN = $(shell go env GOPATH)/bin
endif

.PHONY: all
all: lint test

.PHONY: test
test:
	go test ./...

report: $(GOBIN)/goreportcard-cli
	$(GOBIN)/goreportcard-cli -v

.PHONY: lint
lint: $(GOBIN)/golangci-lint fix-imports
	$(GOBIN)/golangci-lint run --config build/golangci.yaml ./...

.PHONY: fix-imports
fix-imports: $(GOBIN)/goimports
	./build/fix_imports.sh

$(GOBIN)/goreportcard-cli: $(GOBIN)/misspell
	( \
		set -e; \
		d=$(shell mktemp -d); cd $$d; \
		git clone https://github.com/gojp/goreportcard.git; \
		cd goreportcard; \
		make install; \
		go install ./cmd/goreportcard-cli; \
		cd; rm -rf $$d \
	)

$(GOBIN)/goimports:
	./build/go_tool_install.sh goimports

$(GOBIN)/golangci-lint:
	./build/go_tool_install.sh golangci-lint

$(GOBIN)/misspell:
	go get github.com/client9/misspell/cmd/misspell
	./build/go_tool_install.sh misspell

.PHONY: conch
conch: $(GOBIN)/conch
$(GOBIN)/conch:
	./build/go_tool_install.sh conch

.PHONY: clean
clean:
	go clean -testcache
	rm -f $(GOBIN)/goimports
	rm -f $(GOBIN)/golangci-lint
	rm -f $(GOBIN)/goreportcard-cli
	rm -f $(GOBIN)/misspell
	rm -f $(GOBIN)/stringer
	rm -f $(GOBIN)/conch

.PHONY: clean
coverage:
	go test -v -cover -coverprofile=/tmp/coverage.out
	go tool cover -html=/tmp/coverage.out
