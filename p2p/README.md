# Lumera P2P Kademlia Implementation

This directory contains the implementation of the Kademlia Distributed Hash Table (DHT) used by the Lumera Supernode for peer-to-peer communication and distributed data storage.

## Overview

The implementation is based off of a combination of the original [Kademlia whitepaper](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf) and the [xlattice design specification](https://xlattice.sourceforge.net/components/protocol/kademlia/specs.html). It does not attempt to conform to BEP-5, or any other BitTorrent-specific design.

Kademlia is a distributed hash table that provides efficient lookup operations through a XOR-based metric topology. It enables nodes to store and retrieve data in a decentralized network without relying on a central server.

## Key Parameters

The Kademlia implementation uses the following key parameters:

| Parameter | Value | Description |
|-----------|-------|-------------|
| Alpha (α) | 6 | The degree of parallelism in network calls. This determines how many nodes are contacted simultaneously during lookup operations. |
| K | 20 | The maximum number of contacts stored in a bucket. This is the replication parameter that ensures redundancy in the network. |
| B | 256 | The size in bits of the keys used to identify nodes and store/retrieve data (using SHA3-256). |

## Core Components

### DHT (Distributed Hash Table)

The DHT is the main component that manages the distributed storage and retrieval of data. It handles:
- Storing data across the network
- Retrieving data from the network
- Managing node connections and routing

### HashTable

The HashTable maintains the routing table of known peers in the network. It organizes nodes into K-buckets based on their distance from the local node (using XOR metric).

### Network

Handles peer connections, messaging, and encryption. It uses ALTS (Application Layer Transport Security) for secure communication between nodes.

### Store

Provides persistent storage for data in the DHT. The implementation uses SQLite for local storage.

## Key Operations

### Bootstrap Process

When a node starts:
1. It checks for configured bootstrap nodes
2. If none provided, queries the Lumera blockchain for active supernodes
3. Connects to bootstrap nodes and performs iterative `FIND_NODE` queries
4. Builds its routing table based on responses
5. Becomes a full participant in the network

### Data Replication

Data stored in the network is:
1. Stored locally in SQLite
2. Replicated to the closest `Alpha` (6) nodes in the DHT
3. Periodically checked and re-replicated as nodes come and go

### Node Identification

- Node IDs are derived from Lumera accounts in the keyring
- This provides cryptographic identity and authentication between peers
- Keys are base58-encoded Blake3 hashes of the data

## Implementation Details

- The implementation uses a modified Kademlia with `Alpha=6` for parallelism
- K-buckets are sorted by least recently seen nodes
- The routing table consists of 256 buckets, each containing up to 20 nodes
- Periodic refreshing of buckets ensures the routing table stays up-to-date