package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gannoncombs/diyBlockchain/bcrypto"
	"github.com/gannoncombs/diyBlockchain/consensus"
	"github.com/gannoncombs/diyBlockchain/core"
	"github.com/gannoncombs/diyBlockchain/network"
	"github.com/gannoncombs/diyBlockchain/persistence"
)

// --- status ---

func cmdStatus(store *persistence.Store, cfg config) {
	bc := store.Chain()
	state := store.State()
	latest := bc.LatestBlock()

	validatorCount := 0
	for _, stake := range state.StakedBalances {
		if stake >= consensus.MinStake {
			validatorCount++
		}
	}

	if cfg.jsonOutput {
		out, _ := json.MarshalIndent(map[string]interface{}{
			"height":      len(bc.Blocks),
			"latest_hash": latest.Hash,
			"validators":  validatorCount,
		}, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Printf("Chain height:  %d blocks\n", len(bc.Blocks))
	fmt.Printf("Latest hash:   %s\n", latest.Hash)
	fmt.Printf("Validators:    %d\n", validatorCount)
}

// --- balances ---

func cmdBalances(store *persistence.Store, cfg config) {
	state := store.State()

	if cfg.jsonOutput {
		out, _ := json.MarshalIndent(map[string]interface{}{
			"balances":       state.Balances,
			"staked_balances": state.StakedBalances,
		}, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Printf("%-14s %10s %10s\n", "Account", "Spendable", "Staked")
	fmt.Printf("%-14s %10s %10s\n", "-------", "---------", "------")
	for acct, bal := range state.Balances {
		staked := state.StakedBalances[acct]
		if bal > 0 || staked > 0 {
			fmt.Printf("%-14s %10d %10d\n", acct, bal, staked)
		}
	}
}

// --- blocks ---

func cmdBlocks(store *persistence.Store, cfg config) {
	bc := store.Chain()

	if cfg.jsonOutput {
		out, _ := json.MarshalIndent(bc.Blocks, "", "  ")
		fmt.Println(string(out))
		return
	}

	for _, b := range bc.Blocks {
		validator := b.Validator
		if validator == "" {
			validator = "(none)"
		}
		fmt.Printf("Block %-4d  %s  validator=%-10s  txs=%d\n",
			b.Index, b.Hash[:16]+"...", validator, len(b.Transactions))
	}
}

// --- block ---

func cmdBlock(store *persistence.Store, cfg config) {
	fs := flag.NewFlagSet("block", flag.ExitOnError)
	index := fs.Int("index", -1, "Block index to display")
	fs.Parse(cfg.args)

	if *index < 0 {
		fmt.Fprintln(os.Stderr, "Usage: blockchain block --index <N>")
		os.Exit(1)
	}

	bc := store.Chain()
	if *index >= len(bc.Blocks) {
		fmt.Fprintf(os.Stderr, "Block %d not found (chain height: %d)\n", *index, len(bc.Blocks))
		os.Exit(1)
	}

	b := bc.Blocks[*index]

	if cfg.jsonOutput {
		out, _ := json.MarshalIndent(b, "", "  ")
		fmt.Println(string(out))
		return
	}

	validator := b.Validator
	if validator == "" {
		validator = "(none)"
	}

	fmt.Printf("Index:     %d\n", b.Index)
	fmt.Printf("Timestamp: %d\n", b.Timestamp)
	fmt.Printf("Validator: %s\n", validator)
	fmt.Printf("PrevHash:  %s\n", b.PrevHash)
	fmt.Printf("Hash:      %s\n", b.Hash)
	fmt.Printf("Transactions (%d):\n", len(b.Transactions))
	for i, tx := range b.Transactions {
		from := tx.From
		if from == "" {
			from = "COINBASE"
		}
		fmt.Printf("  %d. [%s] %s -> %s : %d\n", i+1, tx.Type, from, tx.To, tx.Amount)
	}
}

// --- send ---

func cmdSend(store *persistence.Store, cfg config) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	from := fs.String("from", "", "Sender wallet address")
	to := fs.String("to", "", "Recipient address")
	amount := fs.String("amount", "0", "Amount to send")
	wallet := fs.String("wallet", "", "Path to wallet file (auto-detected from --from if omitted)")
	fs.Parse(cfg.args)

	if *from == "" || *to == "" || *amount == "0" {
		fmt.Fprintln(os.Stderr, "Usage: blockchain send --from <addr> --to <addr> --amount <n> [--wallet <path>]")
		os.Exit(1)
	}

	amt, err := strconv.ParseUint(*amount, 10, 64)
	if err != nil {
		exitErr("amount", fmt.Errorf("invalid number: %s", *amount))
	}

	tx := core.NewTransaction(*from, *to, amt)

	// Sign the transaction if a wallet is available
	w := loadWalletFor(*from, *wallet, cfg)
	if w != nil {
		if err := bcrypto.SignTransaction(&tx, w.PrivateKey); err != nil {
			exitErr("send", err)
		}
	}

	block, err := mintBlockWithTxs(store, []core.Transaction{tx})
	if err != nil {
		exitErr("send", err)
	}

	fmt.Printf("Sent %d tokens from %s to %s\n", amt, *from, *to)
	fmt.Printf("Included in block %d [%s]\n", block.Index, block.Hash[:16]+"...")
}

// --- stake ---

func cmdStake(store *persistence.Store, cfg config) {
	fs := flag.NewFlagSet("stake", flag.ExitOnError)
	from := fs.String("from", "", "Account to stake from")
	amount := fs.String("amount", "0", "Amount to stake")
	wallet := fs.String("wallet", "", "Path to wallet file")
	fs.Parse(cfg.args)

	if *from == "" || *amount == "0" {
		fmt.Fprintln(os.Stderr, "Usage: blockchain stake --from <addr> --amount <n> [--wallet <path>]")
		os.Exit(1)
	}

	amt, err := strconv.ParseUint(*amount, 10, 64)
	if err != nil {
		exitErr("amount", fmt.Errorf("invalid number: %s", *amount))
	}

	tx := core.NewStakeTransaction(*from, amt)

	w := loadWalletFor(*from, *wallet, cfg)
	if w != nil {
		if err := bcrypto.SignTransaction(&tx, w.PrivateKey); err != nil {
			exitErr("stake", err)
		}
	}

	block, err := mintBlockWithTxs(store, []core.Transaction{tx})
	if err != nil {
		exitErr("stake", err)
	}

	fmt.Printf("Staked %d tokens for %s\n", amt, *from)
	fmt.Printf("Included in block %d [%s]\n", block.Index, block.Hash[:16]+"...")
}

// --- unstake ---

func cmdUnstake(store *persistence.Store, cfg config) {
	fs := flag.NewFlagSet("unstake", flag.ExitOnError)
	from := fs.String("from", "", "Account to unstake from")
	amount := fs.String("amount", "0", "Amount to unstake")
	wallet := fs.String("wallet", "", "Path to wallet file")
	fs.Parse(cfg.args)

	if *from == "" || *amount == "0" {
		fmt.Fprintln(os.Stderr, "Usage: blockchain unstake --from <addr> --amount <n> [--wallet <path>]")
		os.Exit(1)
	}

	amt, err := strconv.ParseUint(*amount, 10, 64)
	if err != nil {
		exitErr("amount", fmt.Errorf("invalid number: %s", *amount))
	}

	tx := core.NewUnstakeTransaction(*from, amt)

	w := loadWalletFor(*from, *wallet, cfg)
	if w != nil {
		if err := bcrypto.SignTransaction(&tx, w.PrivateKey); err != nil {
			exitErr("unstake", err)
		}
	}

	block, err := mintBlockWithTxs(store, []core.Transaction{tx})
	if err != nil {
		exitErr("unstake", err)
	}

	fmt.Printf("Unstaked %d tokens for %s\n", amt, *from)
	fmt.Printf("Included in block %d [%s]\n", block.Index, block.Hash[:16]+"...")
}

// --- validators ---

func cmdValidators(store *persistence.Store, cfg config) {
	state := store.State()

	type valInfo struct {
		Address string `json:"address"`
		Stake   uint64 `json:"stake"`
	}

	var validators []valInfo
	for addr, stake := range state.StakedBalances {
		if stake >= consensus.MinStake {
			validators = append(validators, valInfo{addr, stake})
		}
	}

	if cfg.jsonOutput {
		out, _ := json.MarshalIndent(validators, "", "  ")
		fmt.Println(string(out))
		return
	}

	if len(validators) == 0 {
		fmt.Printf("No validators (minimum stake: %d)\n", consensus.MinStake)
		return
	}

	fmt.Printf("%-14s %10s\n", "Validator", "Stake")
	fmt.Printf("%-14s %10s\n", "---------", "-----")
	for _, v := range validators {
		fmt.Printf("%-14s %10d\n", v.Address, v.Stake)
	}
}

// --- mint ---

func cmdMint(store *persistence.Store, cfg config) {
	block, err := mintBlockWithTxs(store, nil)
	if err != nil {
		exitErr("mint", err)
	}

	if cfg.jsonOutput {
		out, _ := json.MarshalIndent(block, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Printf("Block %d minted by %s\n", block.Index, block.Validator)
	fmt.Printf("Hash: %s\n", block.Hash)
	fmt.Printf("Reward: %d tokens\n", consensus.BlockReward)
}

// --- bootstrap ---

func cmdBootstrap(store *persistence.Store, cfg config) {
	// Check if chain already has more than genesis
	if len(store.Chain().Blocks) > 1 {
		fmt.Fprintln(os.Stderr, "Chain already bootstrapped. Use send/stake/mint commands.")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	accounts := fs.String("accounts", "alice:10000,bob:5000,carol:2000",
		"Comma-separated account:amount pairs for initial distribution")
	fs.Parse(cfg.args)

	// Parse account distributions
	var distTxs []core.Transaction
	for _, pair := range splitComma(*accounts) {
		parts := splitColon(pair)
		if len(parts) != 2 {
			exitErr("bootstrap", fmt.Errorf("invalid account format: %s (expected name:amount)", pair))
		}
		amt, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			exitErr("bootstrap", fmt.Errorf("invalid amount for %s: %s", parts[0], parts[1]))
		}
		distTxs = append(distTxs, core.NewTransaction("treasury", parts[0], amt))
	}

	// Block 1: distribute tokens (no validator yet — bootstrap phase)
	_, err := store.AddBlock(distTxs, "")
	if err != nil {
		exitErr("bootstrap", err)
	}
	fmt.Println("Distributed tokens from treasury.")

	// Block 2: each recipient stakes half their tokens
	var stakeTxs []core.Transaction
	for _, tx := range distTxs {
		stakeAmt := tx.Amount / 2
		if stakeAmt >= consensus.MinStake {
			stakeTxs = append(stakeTxs, core.NewStakeTransaction(tx.To, stakeAmt))
		}
	}

	if len(stakeTxs) > 0 {
		_, err = store.AddBlock(stakeTxs, "")
		if err != nil {
			exitErr("bootstrap", err)
		}
		fmt.Printf("Staked tokens for %d validators.\n", len(stakeTxs))
	}

	fmt.Println("Bootstrap complete!")
}

func splitComma(s string) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func splitColon(s string) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	parts = append(parts, current)
	return parts
}

// --- wallet-new ---

func cmdWalletNew(cfg config) {
	w, err := bcrypto.NewWallet()
	if err != nil {
		exitErr("wallet", err)
	}

	walletDir := filepath.Join(cfg.dataDir, "wallets")
	if err := w.Save(walletDir); err != nil {
		exitErr("wallet", err)
	}

	if cfg.jsonOutput {
		out, _ := json.MarshalIndent(map[string]string{
			"address": w.Address,
			"file":    filepath.Join(walletDir, w.Address+".json"),
		}, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Printf("New wallet created!\n")
	fmt.Printf("  Address: %s\n", w.Address)
	fmt.Printf("  Saved to: %s\n", filepath.Join(walletDir, w.Address+".json"))
}

// --- wallets ---

func cmdWallets(cfg config) {
	walletDir := filepath.Join(cfg.dataDir, "wallets")
	addrs, err := bcrypto.ListWallets(walletDir)
	if err != nil {
		exitErr("wallets", err)
	}

	if cfg.jsonOutput {
		out, _ := json.MarshalIndent(addrs, "", "  ")
		fmt.Println(string(out))
		return
	}

	if len(addrs) == 0 {
		fmt.Println("No wallets found. Use 'blockchain wallet-new' to create one.")
		return
	}

	fmt.Println("Wallets:")
	for _, addr := range addrs {
		fmt.Printf("  %s\n", addr)
	}
}

// --- run ---

func cmdRun(store *persistence.Store, cfg config) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	port := fs.String("port", "3000", "Port to listen on")
	address := fs.String("address", "", "Validator address (wallet address)")
	peers := fs.String("peers", "", "Comma-separated peer URLs (e.g., http://localhost:3001)")
	fs.Parse(cfg.args)

	if *address == "" {
		fmt.Fprintln(os.Stderr, "Usage: blockchain run --address <wallet-addr> [--port 3000] [--peers url1,url2]")
		os.Exit(1)
	}

	node := network.NewNode(*port, *address, store, core.DefaultGenesis())

	// Register seed peers
	if *peers != "" {
		for _, peer := range splitComma(*peers) {
			node.AddPeer(peer)
			// Tell the peer about us
			ourURL := fmt.Sprintf("http://localhost:%s", *port)
			network.RegisterWithPeer(peer, ourURL)
		}
	}

	if err := node.Start(); err != nil {
		exitErr("run", err)
	}
}

// --- helpers ---

// loadWalletFor tries to load a wallet for the given address.
// If an explicit wallet path is given, uses that. Otherwise, looks in datadir/wallets/.
// Returns nil (not an error) if no wallet is found — unsigned mode.
func loadWalletFor(address, walletPath string, cfg config) *bcrypto.Wallet {
	if walletPath != "" {
		w, err := bcrypto.LoadWallet(walletPath)
		if err != nil {
			exitErr("wallet", err)
		}
		return w
	}

	// Auto-detect from datadir
	autoPath := filepath.Join(cfg.dataDir, "wallets", address+".json")
	if _, err := os.Stat(autoPath); err == nil {
		w, err := bcrypto.LoadWallet(autoPath)
		if err != nil {
			exitErr("wallet", err)
		}
		return w
	}

	return nil
}

// mintBlockWithTxs selects a validator via PoS, prepends the block reward,
// and produces a new block containing the given transactions.
func mintBlockWithTxs(store *persistence.Store, userTxs []core.Transaction) (core.Block, error) {
	state := store.State()
	seed := store.Chain().LatestBlock().Hash

	validator, err := consensus.SelectValidator(state, seed)
	if err != nil {
		return core.Block{}, err
	}

	txs := []core.Transaction{consensus.CreateBlockReward(validator)}
	txs = append(txs, userTxs...)

	return store.AddBlock(txs, validator)
}
