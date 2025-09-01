.PHONY: build build-release build-sncli build-sn-manager
.PHONY: install-lumera setup-supernodes system-test-setup 
.PHONY: gen-cascade gen-supernode
.PHONY: test-e2e test-unit test-integration test-system

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Linker flags for version information
LDFLAGS = -X github.com/LumeraProtocol/supernode/v2/supernode/cmd.Version=$(VERSION) \
          -X github.com/LumeraProtocol/supernode/v2/supernode/cmd.GitCommit=$(GIT_COMMIT) \
          -X github.com/LumeraProtocol/supernode/v2/supernode/cmd.BuildTime=$(BUILD_TIME)

# Linker flags for sn-manager
SN_MANAGER_LDFLAGS = -X main.Version=$(VERSION) \
                     -X main.GitCommit=$(GIT_COMMIT) \
                     -X main.BuildTime=$(BUILD_TIME)

build:
	@mkdir -p release
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=amd64 \
	echo "Building supernode..."
	go build \
		-trimpath \
		-ldflags="-s -w $(LDFLAGS)" \
		-o release/supernode-linux-amd64 \
		./supernode
	@chmod +x release/supernode-linux-amd64
	@echo "supernode built successfully at release/supernode-linux-amd64"

build-sncli: release/sncli

SNCLI_SRC := $(wildcard cmd/sncli/*.go) \
             $(wildcard cmd/sncli/**/*.go)

release/sncli: $(SNCLI_SRC) cmd/sncli/go.mod cmd/sncli/go.sum
	@mkdir -p release
	@echo "Building sncli..."
	@RELEASE_DIR=$(CURDIR)/release && \
	cd cmd/sncli && \
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=amd64 \
	go build \
		-trimpath \
		-ldflags="-s -w $(LDFLAGS)" \
		-o $$RELEASE_DIR/sncli && \
	chmod +x $$RELEASE_DIR/sncli && \
	echo "sncli built successfully at $$RELEASE_DIR/sncli"

build-sn-manager:
	@mkdir -p release
	@echo "Building sn-manager..."
	@cd sn-manager && \
	CGO_ENABLED=0 \
	GOOS=linux \
	GOARCH=amd64 \
	go build \
		-trimpath \
		-ldflags="-s -w $(SN_MANAGER_LDFLAGS)" \
		-o ../release/sn-manager \
		.
	@chmod +x release/sn-manager
	@echo "sn-manager built successfully at release/sn-manager"

test-unit:
	go test -v ./...

test-integration:
	go test -v -p 1 -count=1 -tags=integration ./...

test-system:
	cd tests/system && go test -tags=system_test -v .

gen-cascade:
	protoc \
		--proto_path=proto \
		--go_out=gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=gen \
		--go-grpc_opt=paths=source_relative \
		proto/supernode/action/cascade/service.proto

gen-supernode:
	protoc \
		--proto_path=proto \
		--proto_path=$$(go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway)/third_party/googleapis \
		--go_out=gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=gen \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=gen \
		--grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=gen \
		proto/supernode/supernode.proto

# Define the paths
SUPERNODE_SRC=supernode/main.go
DATA_DIR=tests/system/supernode-data1
DATA_DIR2=tests/system/supernode-data2
DATA_DIR3=tests/system/supernode-data3
CONFIG_FILE=tests/system/config.test-1.yml
CONFIG_FILE2=tests/system/config.test-2.yml
CONFIG_FILE3=tests/system/config.test-3.yml

# Setup script
SETUP_SCRIPT=tests/scripts/setup-supernodes.sh

# Install Lumera
# Optional: specify lumera binary path to skip download
LUMERAD_BINARY ?=
# Optional: specify installation mode (latest-release, latest-tag, or vX.Y.Z)
INSTALL_MODE ?=latest-tag

install-lumera:
	@echo "Installing Lumera..."
	@chmod +x tests/scripts/install-lumera.sh
	@sudo LUMERAD_BINARY="$(LUMERAD_BINARY)" tests/scripts/install-lumera.sh $(INSTALL_MODE)
	@echo "PtTDUHythfRfXHh63yzyiGDid4TZj2P76Zd,18749999981413" > ~/claims.csv
# Setup supernode environments
setup-supernodes:
	@echo "Setting up all supernode environments..."
	@chmod +x $(SETUP_SCRIPT)
	@bash $(SETUP_SCRIPT) all $(SUPERNODE_SRC) $(DATA_DIR) $(CONFIG_FILE) $(DATA_DIR2) $(CONFIG_FILE2) $(DATA_DIR3) $(CONFIG_FILE3)

# Complete system test setup (Lumera + Supernodes)
system-test-setup: install-lumera setup-supernodes
	@echo "System test environment setup complete."
	@if [ -f claims.csv ]; then cp claims.csv ~/; echo "Copied claims.csv to home directory."; fi

# Run system tests with complete setup
test-e2e:
	@echo "Running system tests..."
	@cd tests/system && go test -tags=system_test -v .

# Run cascade e2e tests only
test-cascade:
	@echo "Running cascade e2e tests..."
	@cd tests/system && go test -tags=system_test -v -run TestCascadeE2E .

# Run sn-manager e2e tests only
test-sn-manager:
	@echo "Running sn-manager e2e tests..."
	@cd tests/system && go test -tags=system_test -v -run '^TestSNManager' .

# Release command: push branch, tag, and push tag with auto-increment
release:
	@echo "Getting current branch..."
	$(eval CURRENT_BRANCH := $(shell git branch --show-current))
	@echo "Current branch: $(CURRENT_BRANCH)"
	
	@echo "Getting latest tag..."
	$(eval LATEST_TAG := $(shell git tag -l "v*" | sort -V | tail -n1))
	$(eval NEXT_TAG := $(shell \
		if [ -z "$(LATEST_TAG)" ]; then \
			echo "v2.5.0"; \
		else \
			echo "$(LATEST_TAG)" | sed 's/^v//' | awk -F. '{print "v" $$1 "." $$2 "." $$3+1}'; \
		fi))
	@echo "Next tag: $(NEXT_TAG)"
	
	@echo "Pushing branch to upstream..."
	git push upstream $(CURRENT_BRANCH) -f
	
	@echo "Creating and pushing tag $(NEXT_TAG)..."
	git tag $(NEXT_TAG)
	git push upstream $(NEXT_TAG)
	
	@echo "Release complete: $(NEXT_TAG) pushed to upstream"


download-logs:
	@echo "Downloading supernode logs from remote server..."
	@scp root@64.227.183.166:~/runner/supernode.log .

