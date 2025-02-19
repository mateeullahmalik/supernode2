# Simple test targets
.PHONY: test test-unit test-integration test-system

# Run unit tests (regular tests with code)
test-unit:
	go test -v ./...

# Run integration tests
test-integration:
	go test -v -p 1 -count=1 -tags=integration ./...

# Run system tests
test-system:
	go test -v -tags=system ./...

# Run all tests
test: test-unit test-integration test-system