# Phase 3: Proof of Stake Consensus

## Goal
Implement a Proof of Stake consensus mechanism where validators are selected based on their staked tokens.

## What You'll Learn
- How PoS differs from PoW (no mining puzzles, validators are chosen by stake)
- Validator selection algorithms (weighted random by stake)
- Staking and unstaking mechanics
- Block rewards and how new tokens enter circulation
- Go: randomness, sorting, more complex struct relationships

## Background: PoS in Plain English
In Proof of Work, miners race to solve a math puzzle — first to solve it gets to add the next block. This wastes enormous energy.

In Proof of Stake, validators lock up ("stake") their tokens as collateral. The protocol selects a validator to propose the next block, weighted by how much they've staked. If they propose a valid block, they earn a reward. If they cheat, they lose their stake ("slashing").

Think of it like a raffle — more tickets (stake) = higher chance of being picked, but cheating gets you kicked out.

## Requirements

### Validator struct
- `Address` — account identifier
- `Stake` — amount of tokens staked

### Staking
- Special transaction type: `Stake` and `Unstake`
- Staking locks tokens (removed from spendable balance)
- Unstaking returns tokens (with a delay in real systems, instant for simplicity here)
- Minimum stake threshold to become a validator

### Validator Selection
- Weighted random selection proportional to stake
- Deterministic given the same seed (use previous block hash as seed)
- Only validators with stake above minimum threshold are eligible

### Block Proposal
- Selected validator creates the next block
- Block includes validator's address
- Validator receives block reward (coinbase transaction)

### Slashing (simplified)
- If a block fails validation, the proposer's stake is reduced
- Keeps validators honest

## Deliverables
- `consensus/pos.go` — Validator selection algorithm
- `consensus/staking.go` — Stake/unstake logic
- `consensus/pos_test.go` — Tests for selection fairness and determinism
- Update `core/transaction.go` for stake/unstake transaction types
- Update `core/state.go` to track staked balances separately

## Success Criteria
- Validators are selected proportionally to their stake
- Same inputs produce same validator selection (deterministic)
- Block rewards are distributed to the selected validator
- Invalid blocks result in stake slashing
