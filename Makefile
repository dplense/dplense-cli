BINARY_NAME = dplense
MODULE = github.com/dplense/dplense-cli

.PHONY: build test lint clean

build:
	go build -o $(BINARY_NAME) ./cmd/dplense

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY_NAME)

.DEFAULT_GOAL := build
