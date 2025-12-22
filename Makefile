.PHONY: test coverage help

# Run all tests
test:
	go test -v -race ./...

# Run tests with coverage and generate HTML report
coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"
	@go tool cover -func=coverage.out | grep total

# Show help
help:
	@echo "Available targets:"
	@echo "  test      - Run all tests"
	@echo "  coverage  - Run tests with coverage and generate HTML report"
	@echo "  help      - Show this help message"

