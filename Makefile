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
	go test ./...

## fmt: Format all Go files
fmt:
	go fmt ./...

.PHONY: build run clean test fmt
