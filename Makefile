SHELL=/usr/bin/env bash

build:
	GOOS=linux GOARCH=amd64 go build -o main ./cmd
	zip main.zip main

run:
	go run ./cmd run

clean:
	go clean
	rm -f kyc-checks

fmt:
	go fmt ./...
	gofumpt -w .

lint:
	golangci-lint run

test:
	go test -p 4 -v ./...

.PHONY: build run clean test
