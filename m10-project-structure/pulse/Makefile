.PHONY: build test lint tidy run-customer run-profile run-notify

build:
	go build ./...

test:
	go test -race ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

run-customer:
	go run ./cmd/customer-api

run-profile:
	go run ./cmd/profile-svc

run-notify:
	go run ./cmd/notification-svc
