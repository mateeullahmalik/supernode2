# P2P Connection Pool – Design, Usage, and Pitfalls

This document explains the connection pooling and secure transport integration used by the P2P (Kademlia) networking layer. It covers the data structures, concurrency guarantees, eviction policy, failure modes, and recommended improvements to minimize hard‑to‑debug edge cases.

Applies to:
- `p2p/kademlia/conn_pool.go`
- `p2p/kademlia/network.go`

## Goals

- Reuse secure client connections to reduce handshake/dial overhead.
- Provide per‑RPC timeouts and strict read/write deadlines.
- Avoid deadlocks or data races under concurrent access.
- Keep implementation simple and predictable while remaining robust under churn.

## Components

- `ConnPool`
  - `capacity int`: max number of pooled connections (default 256).
  - `conns map[string]*connectionItem`: keyed by a canonical address, e.g. `<identity>@<ip:port>`.
  - `mtx sync.Mutex`: guards the map and capacity/eviction logic.

- `connectionItem`
  - `lastAccess time.Time`: updated on `Get`, used to evict the least‑recently used item.
  - `conn net.Conn`: pooled (wrapped) connection.

- `connWrapper`
  - Wraps the secure (handshaked) connection, embeds a small mutex to serialize concurrent reads/writes across RPCs on the same socket.
  - Implements `net.Conn` and forwards all methods.

- `Network`
  - Owns a `ConnPool` instance.
  - Implements `Call(ctx, request, isLong)` which uses the pool, sets per‑RPC deadlines, and evicts broken connections.
  - Provides a secure server listener (`NewSecureServerConn`) with per‑message read/write deadlines.

## Addressing and Identity

Connections are keyed by a canonical string that embeds the remote identity:

```
<bech32-identity>@<host>:<port>
```

This key matters for:
- Selecting proper credentials for the handshake (the identity is set on a per-connection clone of the transport credentials).
- Ensuring all RPCs to the same remote reuse the same socket.

## Secure Handshake Flow (Client)

`NewSecureClientConn(ctx, tc, remoteAddr)`:
1. Parse `<bech32>@ip:port` into `(remoteIdentity, remoteAddress)`.
2. Clone credentials and set `remoteIdentity` on the clone.
3. Dial TCP with a short timeout (3s) and keepalive (30s).
4. Perform the client handshake on the cloned credentials.
5. Return a `connWrapper{secureConn, rawConn}` ready for pooling.

Notes:
- No global deadlines are set; `Network.Call` applies per‑RPC deadlines.
- If handshake fails, the raw socket is closed immediately.

## Server Handling

`Network.handleConn` wraps the accepted socket with server credentials. For each request:
- Enforces a per‑message read deadline (currently 60s) to avoid slow peers hanging the server.
- Decodes message → handles → encodes response → enforces write deadline.

## Network.Call – Using the Pool

High‑level flow:
1. Compute `remoteAddr` key `<id>@<ip:port>`.
2. Try `pool.Get(remoteAddr)`.
3. On miss: create a new secure connection with `NewSecureClientConn`, then **double‑check** the pool (to avoid races) and either reuse existing or add the new one.
4. Encode message (once). Apply per‑RPC write/read deadlines. Perform the RPC on the pooled connection.
5. On error during write/read/deadline, evict from pool and close.

Concurrency:
- Each `connWrapper` has an internal mutex to serialize RPCs over the same socket, avoiding interleaved reads/writes.
- Pool map is protected by a separate mutex.

## Eviction Policy

- **Capacity‑based eviction**: When adding a connection and the pool is full, the oldest (by `lastAccess`) is closed and removed.
- **Replacement**: If adding a connection for an address that already exists, the old connection is closed and replaced without counting as a capacity increase.
- **Error‑based eviction**: If an RPC write/read fails on a pooled connection, the pool evicts that entry and closes the socket.

### Idle Eviction (Disabled)

To keep behavior simple and predictable, there is no background idle eviction. Connections remain in the pool until they fail during an RPC (error-based eviction) or a new connection is added while the pool is at capacity (LRU by `lastAccess`). If proactive cleanup of idle sockets is desired, add it behind a feature flag.

### Known Edge Case: Eviction While In‑Use

If a capacity eviction closes a connection that another goroutine is currently using, that in‑flight RPC will fail. This is rare but possible under heavy churn when many new addresses are added. Current `Network.Call` behavior treats this as a normal network failure and retries will establish a fresh connection.

Mitigations (trade‑offs):
- Keep pool capacity conservative to reduce churn‑induced eviction.
- Reintroduce optional idle eviction (e.g., close conns idle > 10–15 min) to free stale entries proactively.
- Add an optional "in‑use" refcount to avoid evicting sockets with ongoing RPCs (heavier change; requires marking usage around `Call`).

## Timeouts & Deadlines

- Message type timeouts: see `network.go`, `execTimeouts`, and `getExecTimeout` for per‑type values.
- Client per‑RPC deadlines (write ~3s; read ~message‑specific timeout) are set around each `Call`.
- Server per‑message read/write deadlines (60s) apply to each request loop iteration.

## Failure Modes & Handling

- Dial/Handshake failure: `NewSecureClientConn` returns error; no pool side‑effects.
- Write/Read failure during RPC: The pooled connection is evicted and closed (`mustDrop` path for wrapper; similar for non‑wrapper fallback). The caller receives an error and may retry.
- Invalid addressing (e.g., bad ports): `Call` performs quick sanity checks and fails early.

## Thread Safety Summary

- `ConnPool` map is guarded by `mtx`.
- Each pooled connection (`connWrapper`) serializes concurrent reads/writes via an internal mutex.
- `Network.Call` protects pool add/get/del with a separate mutex to avoid races.
- Eviction always occurs **after** releasing the `connWrapper` lock to prevent deadlocks on `Close()`.

## Replacement Logic – Important Detail

When adding a new connection for an address that already exists, the pool:
- Closes the old connection.
- Overwrites the map entry **without** running capacity eviction.

This avoids spurious eviction of an unrelated (oldest) connection during in‑place replacement.

## Best Practices for Callers

- Always go through `Network.Call` – do not write to pooled connections directly.
- Assume that a pooled connection may be evicted at any time due to errors; handle and surface errors cleanly.
- Do not hold onto `net.Conn` references beyond the scope of a single RPC.

## Suggested Improvements (Roadmap)

1. Idle eviction (optional): Periodic purge of connections idle for > N minutes, to keep the pool healthy without lowering capacity.
2. In‑use awareness: Track refcounts or a simple "in use" flag to prefer evicting idle connections first.
3. Metrics: Export basic counters (dials, handshakes, reuses, evictions) and latencies to help tune capacity.
4. Configuration: Make pool capacity and idle timeout configurable via `p2p.Config`.
5. Cleanup hooks: On shutdown, `ConnPool.Release()` is called; consider awaiting in‑flight RPCs if a graceful shutdown is needed.

## Quick Reference

- Add connection (miss): `pool.Add(remoteAddr, conn)` → may evict oldest if at capacity.
- Replace connection (same addr): `pool.Add(remoteAddr, conn)` → closes old, overwrites entry, **no** capacity eviction.
- Get connection: `conn, err := pool.Get(remoteAddr)` → updates `lastAccess`.
- Evict on error: `pool.Del(remoteAddr)` after closing.
- Release all: `pool.Release()` – closes sockets and clears the map.

## Code Links

- Pool: [`p2p/kademlia/conn_pool.go`](../p2p/kademlia/conn_pool.go)
- Network integration: [`p2p/kademlia/network.go`](../p2p/kademlia/network.go)
