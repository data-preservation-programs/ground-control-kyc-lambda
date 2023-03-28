SHELL=/usr/bin/env bash

build:
	GOARCH=amd64 go build -o main ./main.go
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
