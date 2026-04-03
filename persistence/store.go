package persistence

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gannoncombs/diyBlockchain/bcrypto"
	"github.com/gannoncombs/diyBlockchain/core"
)

const chainFileName = "blockchain.jsonl"

// Store manages blockchain persistence and maintains an in-memory state
// that's updated incrementally — no full replay needed during normal operation.
type Store struct {
	dataDir        string
	genesis        core.Genesis
	chain          *core.Blockchain
	state          *core.State
	file           *os.File // Kept open for appending new blocks
	verifySigs     bool     // When true, all non-coinbase transactions must be signed
}

// NewStore opens or creates a blockchain store in the given directory.
// If a chain file exists, it loads and validates it. Otherwise, starts fresh.
func NewStore(dataDir string, genesis core.Genesis) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	s := &Store{
		dataDir: dataDir,
		genesis: genesis,
	}

	chainPath := filepath.Join(dataDir, chainFileName)

	if _, err := os.Stat(chainPath); os.IsNotExist(err) {
		// No existing chain — start fresh
		return s.initNew(chainPath)
	}

	// Load existing chain
	return s, s.load(chainPath)
}

// initNew creates a fresh blockchain and writes the genesis block to disk.
func (s *Store) initNew(chainPath string) (*Store, error) {
	s.chain = core.NewBlockchain()

	// Build initial state from genesis
	state, err := core.NewStateFromChain(s.genesis, s.chain)
	if err != nil {
		return nil, fmt.Errorf("building initial state: %w", err)
	}
	s.state = state

	// Open file for writing and persist the genesis block
	f, err := os.Create(chainPath)
	if err != nil {
		return nil, fmt.Errorf("creating chain file: %w", err)
	}
	s.file = f

	if err := s.appendBlock(s.chain.Blocks[0]); err != nil {
		return nil, err
	}

	return s, nil
}

// load reads an existing chain from disk, validates it, and rebuilds state.
func (s *Store) load(chainPath string) error {
	f, err := os.Open(chainPath)
	if err != nil {
		return fmt.Errorf("opening chain file: %w", err)
	}

	var blocks []core.Block
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		var block core.Block
		if err := json.Unmarshal(scanner.Bytes(), &block); err != nil {
			return fmt.Errorf("parsing block on line %d: %w", lineNum, err)
		}
		blocks = append(blocks, block)
	}
	f.Close()

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading chain file: %w", err)
	}

	if len(blocks) == 0 {
		return fmt.Errorf("chain file is empty")
	}

	// Rebuild the chain in memory
	s.chain = &core.Blockchain{Blocks: blocks}

	// Validate chain integrity
	if err := s.chain.IsValid(); err != nil {
		return fmt.Errorf("chain validation failed: %w", err)
	}

	// Replay to build state
	state, err := core.NewStateFromChain(s.genesis, s.chain)
	if err != nil {
		return fmt.Errorf("rebuilding state: %w", err)
	}
	s.state = state

	// Reopen file in append mode for future writes
	s.file, err = os.OpenFile(chainPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("reopening chain file for append: %w", err)
	}

	return nil
}

// SetVerifySignatures enables or disables signature verification.
// When enabled, all non-coinbase transactions must have valid signatures.
func (s *Store) SetVerifySignatures(enabled bool) {
	s.verifySigs = enabled
}

// AddBlock validates transactions against current state, appends the block,
// and updates the state incrementally.
func (s *Store) AddBlock(transactions []core.Transaction, validator string) (core.Block, error) {
	// Verify signatures if enabled
	if s.verifySigs {
		for _, tx := range transactions {
			if err := bcrypto.VerifyTransaction(tx); err != nil {
				return core.Block{}, fmt.Errorf("signature verification failed: %w", err)
			}
		}
	}

	// Validate all transactions against current state before committing
	for _, tx := range transactions {
		if err := s.state.Apply(tx); err != nil {
			// State is now inconsistent — rebuild it
			s.rebuildState()
			return core.Block{}, fmt.Errorf("invalid transaction: %w", err)
		}
	}

	// State is already updated. Now persist the block.
	block := s.chain.AddBlock(transactions, validator)

	if err := s.appendBlock(block); err != nil {
		return core.Block{}, err
	}

	return block, nil
}

// appendBlock writes a single block as a JSON line to the chain file.
func (s *Store) appendBlock(block core.Block) error {
	data, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("serializing block %d: %w", block.Index, err)
	}

	if _, err := s.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing block %d: %w", block.Index, err)
	}

	// Sync to disk immediately — durability over speed
	return s.file.Sync()
}

// AcceptBlock validates and appends a pre-built block (received from a peer).
// Unlike AddBlock, it doesn't create the block — it accepts one as-is.
func (s *Store) AcceptBlock(block core.Block) error {
	// Verify it links to our latest block
	latest := s.chain.LatestBlock()
	if block.PrevHash != latest.Hash {
		return fmt.Errorf("block %d doesn't link to our chain (expected prevhash %s, got %s)",
			block.Index, latest.Hash[:16], block.PrevHash[:16])
	}
	if block.Index != latest.Index+1 {
		return fmt.Errorf("block index %d doesn't follow our latest %d", block.Index, latest.Index)
	}

	// Verify block hash
	if block.Hash != block.CalculateHash() {
		return fmt.Errorf("block %d has invalid hash", block.Index)
	}

	// Verify signatures if enabled
	if s.verifySigs {
		for _, tx := range block.Transactions {
			if err := bcrypto.VerifyTransaction(tx); err != nil {
				return fmt.Errorf("signature verification failed: %w", err)
			}
		}
	}

	// Apply transactions to state
	for _, tx := range block.Transactions {
		if err := s.state.Apply(tx); err != nil {
			s.rebuildState()
			return fmt.Errorf("invalid tx in block %d: %w", block.Index, err)
		}
	}

	s.chain.Blocks = append(s.chain.Blocks, block)

	return s.appendBlock(block)
}

// ReplaceChain swaps the current chain with a longer valid one (from a peer).
// Returns an error if the new chain is invalid or not longer.
func (s *Store) ReplaceChain(newBlocks []core.Block) error {
	if len(newBlocks) <= len(s.chain.Blocks) {
		return fmt.Errorf("incoming chain (%d) is not longer than ours (%d)",
			len(newBlocks), len(s.chain.Blocks))
	}

	newChain := &core.Blockchain{Blocks: newBlocks}
	if err := newChain.IsValid(); err != nil {
		return fmt.Errorf("incoming chain is invalid: %w", err)
	}

	// Rebuild state from the new chain
	state, err := core.NewStateFromChain(s.genesis, newChain)
	if err != nil {
		return fmt.Errorf("invalid state in incoming chain: %w", err)
	}

	s.chain = newChain
	s.state = state

	// Rewrite the chain file from scratch
	chainPath := filepath.Join(s.dataDir, chainFileName)
	s.file.Close()

	f, err := os.Create(chainPath)
	if err != nil {
		return fmt.Errorf("rewriting chain file: %w", err)
	}
	s.file = f

	for _, block := range newBlocks {
		if err := s.appendBlock(block); err != nil {
			return err
		}
	}

	return nil
}

// rebuildState replays the entire chain to restore a consistent state.
// Only needed if an Apply call fails partway through a block's transactions.
func (s *Store) rebuildState() {
	state, _ := core.NewStateFromChain(s.genesis, s.chain)
	s.state = state
}

// Chain returns the in-memory blockchain.
func (s *Store) Chain() *core.Blockchain {
	return s.chain
}

// State returns the current state — always up to date, no replay needed.
func (s *Store) State() *core.State {
	return s.state
}

// Close releases the chain file handle.
func (s *Store) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
