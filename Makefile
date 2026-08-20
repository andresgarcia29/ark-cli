.PHONY: test coverage lint build help

# Run all tests with the race detector
test:
	go test -race ./...

# Run tests with coverage and generate an HTML report
coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1

# Vet and format check
lint:
	gofmt -l .
	go vet ./...

build:
	go build -o bin/ark .

help:
	@echo "test      - Run all tests with -race"
	@echo "coverage  - Run tests and write coverage.html"
	@echo "lint      - gofmt check and go vet"
	@echo "build     - Build ./bin/ark"
