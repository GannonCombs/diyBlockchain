# Phase 4: Persistence

## Goal
Save the blockchain to disk and reload it on startup so data survives restarts.

## What You'll Learn
- How blockchains serialize and store data
- File-based storage vs databases
- Reconstructing state from stored blocks
- Go: file I/O, JSON encoding/decoding, `os` package

## Requirements

### Storage Format
- Each block serialized as JSON, one per line (JSON Lines format)
- Single file: `data/blockchain.json`
- Genesis block is always first line

### Save
- Append new blocks to the file as they're created
- Flush to disk after each write (durability)

### Load
- On startup, read the file line by line
- Deserialize each block and rebuild the chain in memory
- Validate the entire chain after loading (detect corruption/tampering)
- If no file exists, start fresh with genesis block

### State Snapshot
- Maintain an incremental state snapshot (balances) updated after each block
- Eliminates the need to replay all transactions from genesis on every query
- Full replay from genesis only needed if snapshot is corrupted or missing
- Inspired by Ethereum's approach: compute state once, update incrementally

### Genesis Config
- `genesis.json` file defines initial balances and chain parameters
- Loaded once at chain creation, never modified

## Deliverables
- `persistence/store.go` — Save/load blockchain to/from disk
- `persistence/store_test.go` — Tests for round-trip serialization
- `core/genesis.go` updated with file-based genesis loading
- `genesis.json` — Default genesis configuration

## Success Criteria
- Chain persists across program restarts
- Corrupted/tampered files are detected on load
- Clean startup from empty state works
