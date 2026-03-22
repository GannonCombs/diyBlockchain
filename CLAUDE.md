# diyBlockchain

A blockchain built from scratch in Go for learning purposes. Implements Proof of Stake consensus.

## Project Goals

1. **Learn blockchain mechanics** at a low level — every component built by hand
2. **Learn Go** along the way — this is a teaching project
3. **Proof of Stake** consensus (not Proof of Work)

## Tech Stack

- **Language**: Go 1.17+
- **No frameworks** for core blockchain logic — only stdlib + minimal dependencies
- **Cobra** for CLI (optional, can use stdlib `flag` to keep it simple)

## Project Structure

```
├── cmd/              # CLI entry points
│   └── blockchain/   # Main binary
├── core/             # Core blockchain types (Block, Blockchain, State)
├── consensus/        # Proof of Stake validator selection, staking
├── crypto/           # Hashing, wallets, digital signatures
├── network/          # P2P networking, node sync
├── persistence/      # Disk storage, chain serialization
├── docs/             # PRDs and design documents
│   └── prd/
└── go.mod
```

## Conventions

- **Explain before building** — this is a learning project, so clarity > cleverness
- **Incremental commits** — each phase gets its own branch and PR
- **Tests for core logic** — especially block validation, consensus, and crypto
- **Go idioms** — exported names are PascalCase, unexported are camelCase, errors returned (not panicked)

## Build & Run

```bash
go build ./cmd/blockchain
go test ./...
```
