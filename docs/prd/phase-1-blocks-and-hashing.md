# Phase 1: Blocks & Hashing

## Goal
Build the foundational data structures — Block and Blockchain — with cryptographic linking via SHA-256.

## What You'll Learn
- What a block actually contains and why
- How SHA-256 hashing creates tamper-proof links between blocks
- How the genesis block bootstraps a chain
- Go: structs, methods, slices, JSON serialization, `crypto/sha256`

## Requirements

### Block struct
- `Index` — position in the chain (0 for genesis)
- `Timestamp` — when the block was created
- `Data` — payload (later replaced by transactions)
- `PrevHash` — hash of the previous block (links the chain)
- `Hash` — this block's own hash (computed from all fields above)

### Hashing
- Block hash = SHA-256 of (Index + Timestamp + Data + PrevHash)
- Any change to any field produces a completely different hash
- This is what makes the chain immutable — alter one block and every hash after it breaks

### Blockchain struct
- Ordered slice of Blocks
- Starts with a genesis block (hardcoded initial block)
- `AddBlock(data)` — creates a new block linked to the previous one
- `IsValid()` — walks the chain verifying every hash and link

## Deliverables
- `core/block.go` — Block struct + hashing
- `core/blockchain.go` — Blockchain struct + genesis + validation
- `core/block_test.go` — Tests for hashing and validation
- Working `main.go` that creates a chain and prints it

## Success Criteria
- Can create a genesis block and append blocks
- Tampering with any block causes `IsValid()` to return false
- All tests pass
