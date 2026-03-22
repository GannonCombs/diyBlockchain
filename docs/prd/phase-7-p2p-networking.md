# Phase 7: P2P Networking

## Goal
Enable multiple nodes to communicate, share blocks, and reach consensus — making the blockchain truly decentralized.

## What You'll Learn
- Why decentralization requires networking
- How nodes discover and talk to each other
- Chain synchronization — how a new node catches up
- Fork resolution — what happens when two validators propose at the same time
- Go: `net/http` or TCP sockets, goroutines, channels, concurrency

## Background: Why P2P?
A blockchain on one machine is just a database. The "block" + "chain" only becomes meaningful when multiple independent nodes maintain copies and agree on the same state. This is the hard part — and the most rewarding to understand.

## Requirements

### Node
- Each node runs an HTTP server (simpler than raw TCP for learning)
- Configurable listen port
- Maintains list of known peer addresses

### Peer Discovery
- Bootstrap with a list of known seed peers
- Nodes share their peer lists with each other
- New nodes can join by connecting to any existing node

### Block Propagation
- When a validator creates a block, broadcast it to all peers
- Peers validate the block and add it to their chain
- Peers forward new blocks to their own peers (gossip protocol)

### Chain Sync
- New node requests the full chain from a peer
- Accepts the longest valid chain (fork resolution)
- Catches up block-by-block

### Endpoints
- `POST /block` — Receive a new block from a peer
- `GET /chain` — Return the full chain (for sync)
- `GET /peers` — Return known peers
- `POST /peers` — Register a new peer
- `POST /tx` — Submit a transaction to the mempool
- `GET /status` — Node status (height, peers, etc.)

## Deliverables
- `network/node.go` — HTTP server, peer management
- `network/sync.go` — Chain synchronization
- `network/node_test.go` — Tests (using multiple local nodes)
- Update CLI with `--port` and `--peers` flags

## Success Criteria
- Can run 3+ nodes locally on different ports
- Blocks propagate to all nodes
- New node syncs full chain from peers
- Nodes converge on the same chain state
