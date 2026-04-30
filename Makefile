BINARY_NAME=krypt
MAIN_PATH=./cmd/krypt/main.go

## build: Compile the binary
build:
	go build -ldflags="-s -w" -buildvcs=false -o bin/$(BINARY_NAME) $(MAIN_PATH)

## run: Build and run the application
run: build
	./bin/$(BINARY_NAME)

## clean: Remove build artifacts
clean:
	go clean
	rm -rf bin/

## test: Run all tests
test:
	go test -race -shuffle=on ./...

## fmt: Format all Go files
fmt:
	golangci-lint fmt ./...

## lint: Run linter
lint:
	golangci-lint run ./...

## lint-fix: Run linter and auto-fix issues
lint-fix:
	golangci-lint run --fix ./...

## setup: Install git hooks and tools
setup: setup-hooks setup-tools

## setup-hooks: Configure git to use .githooks/
setup-hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks configured."

## setup-tools: Install development tools
setup-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: build run clean test fmt lint lint-fix setup setup-hooks setup-tools
