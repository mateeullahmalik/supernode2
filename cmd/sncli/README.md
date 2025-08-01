# sncli - Supernode CLI Client

`sncli` is a lightweight command-line interface for interacting with a Lumera Supernode over secure gRPC. It supports health checks, service discovery, task registration, and status queries.

---

## 🔧 Build Instructions

To build `sncli` from source:

```bash
make build-sncli
```

> The binary will be located at: `release/sncli`

---

## ⚙️ Configuration

Create a `config.toml` file in the same directory where you run `sncli`:

```toml
# Lumera blockchain connection settings
[lumera]
grpc_addr = "localhost:9090"
chain_id = "lumera-devnet-1"

# Keyring settings for managing keys and identities
[keyring]
backend = "test"                     # "file", "test", or "os"
dir = "~/.lumera"                    # Directory where keyring is stored
key_name = "sncli-account"           # Name of local key
local_address = "lumera1abc..."      # Bech32 address of local account (must exist on-chain)

# Supernode peer information
[supernode]
grpc_endpoint = "127.0.0.1:4444"
address = "lumera1supernodeabc123"   # Bech32 address of the Supernode
```

> Ensure the `local_address` exists on-chain (i.e., has received funds or sent a tx).

---

## 🚀 Usage

Run the CLI by calling the built binary with a command:

```bash
./sncli <command> [args...]
```

### Supported Commands

| Command                  | Description                                                  |
|--------------------------|--------------------------------------------------------------|
| `help`                  | Show usage instructions                                      |
| `list`                  | List available gRPC services from the Supernode              |
| `health-check`          | Check if the Supernode is alive                              |
| `get-supernode-status`  | Query current CPU/memory usage reported by the Supernode     |

### Example

```bash
./sncli get-supernode-status
```

---

## 📝 Notes

- `sncli` uses a secure gRPC connection with a handshake based on the Lumera keyring.
- The Supernode address must match a known peer on the network.
- Make sure `sncli-account` has been initialized and exists on the chain.

