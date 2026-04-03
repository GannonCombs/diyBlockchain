package main

import (
	"fmt"
	"os"

	"github.com/gannoncombs/diyBlockchain/core"
	"github.com/gannoncombs/diyBlockchain/persistence"
)

const usage = `diyBlockchain — a blockchain built from scratch

Usage:
  blockchain <command> [flags]

Commands:
  status       Show chain height, latest hash, and validator count
  balances     Show all account balances
  blocks       List all blocks in the chain
  block        Show details of a specific block (--index N)
  send         Send tokens (--from, --to, --amount)
  stake        Stake tokens to become a validator (--from, --amount)
  unstake      Unstake tokens (--from, --amount)
  validators   List current validators and their stakes
  mint         Produce a new block via Proof of Stake
  bootstrap    Set up initial accounts and validators
  wallet-new   Generate a new wallet (key pair)
  wallets      List all wallets
  run          Start a node (--port, --address, --peers)

Global flags:
  --datadir    Data directory (default: ./data)
  --genesis    Path to genesis.json (default: ./genesis.json)
  --json       Output as JSON where supported
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(0)
	}

	// Find the subcommand: first non-flag arg (skipping flag values).
	// Global flags like "--datadir /tmp/x" have a value that follows them.
	globalFlagsWithValue := map[string]bool{"--datadir": true, "--genesis": true}
	var command string
	var args []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if globalFlagsWithValue[arg] {
			// This flag consumes the next arg as its value
			args = append(args, arg)
			if i+1 < len(os.Args) {
				i++
				args = append(args, os.Args[i])
			}
		} else if arg == "--json" {
			args = append(args, arg)
		} else if command == "" {
			command = arg
		} else {
			args = append(args, arg)
		}
	}
	if command == "" {
		fmt.Print(usage)
		os.Exit(0)
	}

	// Parse global flags from the remaining args
	cfg := parseGlobalFlags(args)

	// Load genesis
	genesis, err := loadGenesis(cfg.genesisPath)
	if err != nil {
		exitErr("genesis", err)
	}

	// Open the store
	store, err := persistence.NewStore(cfg.dataDir, genesis)
	if err != nil {
		exitErr("store", err)
	}
	defer store.Close()

	// Dispatch
	switch command {
	case "status":
		cmdStatus(store, cfg)
	case "balances":
		cmdBalances(store, cfg)
	case "blocks":
		cmdBlocks(store, cfg)
	case "block":
		cmdBlock(store, cfg)
	case "send":
		cmdSend(store, cfg)
	case "stake":
		cmdStake(store, cfg)
	case "unstake":
		cmdUnstake(store, cfg)
	case "validators":
		cmdValidators(store, cfg)
	case "mint":
		cmdMint(store, cfg)
	case "bootstrap":
		cmdBootstrap(store, cfg)
	case "wallet-new":
		cmdWalletNew(cfg)
	case "wallets":
		cmdWallets(cfg)
	case "run":
		cmdRun(store, cfg)
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		fmt.Print(usage)
		os.Exit(1)
	}
}

func loadGenesis(path string) (core.Genesis, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No genesis file — use default
		return core.DefaultGenesis(), nil
	}
	return core.LoadGenesis(path)
}

func exitErr(context string, err error) {
	fmt.Fprintf(os.Stderr, "Error (%s): %s\n", context, err)
	os.Exit(1)
}
