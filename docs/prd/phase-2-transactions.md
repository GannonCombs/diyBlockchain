# Phase 2: Transactions & State

## Goal
Replace raw string data in blocks with a proper transaction model. Track account balances.

## What You'll Learn
- How transactions represent value transfer
- How "state" is computed by replaying all transactions in order
- Why validation matters (can't spend more than you have)
- Go: maps, custom types, error handling patterns

## Requirements

### Transaction struct
- `From` — sender account
- `To` — recipient account
- `Amount` — token quantity
- `Timestamp` — when the transaction was created

### State
- `Balances map[string]uint64` — current account balances
- Genesis state defines initial token distribution (e.g., "treasury" starts with 1,000,000 tokens)
- State is computed by replaying all transactions across all blocks in order

### Validation Rules
- Sender must have sufficient balance
- Amount must be greater than 0
- Sender and receiver must be different accounts
- Genesis/coinbase transactions (from = "") are exempt from balance checks (used for staking rewards later)

### Block Update
- Block.Data becomes `[]Transaction` instead of a string
- Hashing now covers the serialized transaction list

## Deliverables
- `core/transaction.go` — Transaction struct + validation
- `core/state.go` — State computation from genesis + chain
- `core/genesis.go` — Genesis configuration
- Update `core/block.go` to use transactions
- Tests for transaction validation and state computation

## Success Criteria
- Can create transactions and pack them into blocks
- Invalid transactions (overspend, zero amount) are rejected
- State correctly reflects all transactions across the chain
