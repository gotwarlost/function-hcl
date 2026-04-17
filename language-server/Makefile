
.PHONY: default
default: build test lint

.PHONY: build
build:
	go install ./...

.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt:
	gofumpt -w .

.PHONY: lint
lint:
	golangci-lint run
