# Simple test targets
.PHONY: test-unit test-integration test-system tests-system-setup

# Run unit tests (regular tests with code)
test-unit:
	go test -v ./...

# Run integration tests
test-integration:
	go test -v -p 1 -count=1 -tags=integration ./...

# Run system tests
test-system:
	cd tests/system && go test -tags=system_test -v .

tests-system-setup:
	cd tests/scripts && ./install-lumera.sh