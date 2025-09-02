# Testing Guide

This repo has two main kinds of tests: integration (unit-ish with multiple components) and system (end‑to‑end with local supernodes + a local chain).

## Quick Start

- P2P integration test (single process):
  - `go test ./tests/integration/p2p -timeout 10m -run ^TestP2PBasicIntegration$ -v`

- System cascade E2E (spawns local supernodes and a local chain):
  - `make setup-supernodes`
  - `make test-cascade`

For a full system suite, use `make test-e2e`.

## Local/Loopback Networking In Tests

Production nodes reject loopback/private addresses when discovering peers. Tests run everything on localhost, so the P2P layer must accept local addresses during tests.

We centralize this behavior behind an environment flag used by the tests themselves:

- `INTEGRATION_TEST=true`
  - When set, the P2P layer allows peers on `127.0.0.1` and hostnames like `localhost`.
  - This applies to neighbor discovery and to the sender address on outbound P2P messages.
  - The normal production behavior (rejecting unspecified/loopback/private addresses) remains unchanged when the flag is not set.

Where this is used:

- `p2p/kademlia/dht.go`
  - `addKnownNodes`: Permits loopback/private/localhost peers when `INTEGRATION_TEST=true`.
  - `newMessage`: Chooses a safe sender IP. In test mode, falls back to `127.0.0.1` when no public IP is available.

You do not need to export `INTEGRATION_TEST` manually for system tests; the tests set/unset it around execution where needed.

## System Test Supernode Layout

System tests use three local supernodes prepared under `tests/system`:

- Configs: `config.test-1.yml`, `config.test-2.yml`, `config.test-3.yml`
  - Supernode listen hosts: `0.0.0.0`
  - P2P ports: `4445`, `4447`, `4449` (paired with gRPC ports `4444`, `4446`, `4448`)
  - Lumera gRPC: `localhost:9090` (local chain started by tests)

- Setup helper: `make setup-supernodes`
  - Builds a `supernode` binary into `tests/system/supernode-data*`
  - Copies the matching yaml config + test keyrings

- Runtime helper: `StartAllSupernodes` in `tests/system/supernode-utils.go`
  - Launches the three supernodes using their config directories

## Troubleshooting

- “unable to fetch bootstrap IP addresses. No valid supernodes found.”
  - Ensure the system tests are starting the local chain and supernodes first (use the Make targets).
  - Confirm no port conflicts on `4444–4449` and that the processes are running.
  - The `INTEGRATION_TEST` flag is already managed by tests; no extra setup required.

- P2P integration test flakiness
  - This test brings up multiple P2P instances in‑process. Give it up to 10 minutes (`-timeout 10m`) on slow machines.

## Design Notes

We intentionally keep the test‑environment override behind a single env var to avoid widening production configuration surface. If we later want to move this into YAML, we can add a boolean like `p2p.allow_local_addresses: true` and thread it into the DHT options—but for now the env‑based switch keeps runtime logic minimal and isolated to tests.

