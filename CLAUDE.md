# diyBlockchain

A blockchain built from scratch in Go for learning purposes. Implements Proof of Stake consensus.

## Project Goals

1. **Learn blockchain mechanics** at a low level — every component built by hand
2. **Learn Go** along the way — this is a teaching project
3. **Proof of Stake** consensus (not Proof of Work)

## Tech Stack

- **Language**: Go 1.17+
- **No frameworks** for core blockchain logic — only stdlib + minimal dependencies
- **stdlib `flag`** for CLI (no Cobra dependency)

## Project Structure

```
├── cmd/blockchain/   # CLI entry point and commands
├── core/             # Block, Blockchain, Transaction, State, Genesis
├── consensus/        # Proof of Stake validator selection, staking, slashing
├── bcrypto/          # ECDSA wallets, digital signatures
├── network/          # P2P node, HTTP API, chain sync, tx gossip
├── persistence/      # JSON Lines storage, incremental state
├── docs/prd/         # PRDs for each phase
├── genesis.json      # Default genesis configuration
└── go.mod
```

## Conventions

- **Explain before building** — this is a learning project, so clarity > cleverness
- **Tests for core logic** — especially block validation, consensus, and crypto
- **Go idioms** — exported names are PascalCase, unexported are camelCase, errors returned (not panicked)

## Build & Run

```bash
go build ./cmd/blockchain
go test ./...
```

## Running a Node

```bash
# Bootstrap a fresh chain
go run ./cmd/blockchain/ bootstrap

# Start a node
go run ./cmd/blockchain/ run --address <wallet-addr> --port 3001 --peers http://localhost:3002

# CLI commands (against local data)
go run ./cmd/blockchain/ status
go run ./cmd/blockchain/ balances
go run ./cmd/blockchain/ send --from <addr> --to <addr> --amount 100
go run ./cmd/blockchain/ wallet-new
```
