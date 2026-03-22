# Phase 5: CLI

## Goal
Build a command-line interface to interact with the blockchain — check balances, send transactions, view blocks, and manage staking.

## What You'll Learn
- How users interact with a blockchain node
- CLI design patterns in Go
- Go: `flag` package or Cobra library, program structure

## Requirements

### Commands
- `blockchain run` — Start the node (loads chain, listens for commands)
- `blockchain balances` — Show all account balances
- `blockchain send --from <addr> --to <addr> --amount <n>` — Create a transaction
- `blockchain blocks` — List all blocks in the chain
- `blockchain block <index>` — Show details of a specific block
- `blockchain stake --from <addr> --amount <n>` — Stake tokens
- `blockchain unstake --from <addr> --amount <n>` — Unstake tokens
- `blockchain validators` — List current validators and their stakes
- `blockchain status` — Show chain height, latest hash, number of validators

### Output
- Human-readable formatted output
- JSON output option (`--json` flag) for programmatic use

## Deliverables
- `cmd/blockchain/main.go` — Entry point
- `cmd/blockchain/commands.go` — Command definitions and handlers
- Integration with persistence layer (auto-save after mutations)

## Success Criteria
- All commands work and produce correct output
- Invalid inputs produce helpful error messages
- Chain state persists between CLI invocations
