package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Block represents a single block in the blockchain.
// Each block contains transactions and is cryptographically linked to the previous block.
type Block struct {
	Index        uint64        `json:"index"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	Validator    string        `json:"validator"`
	PrevHash     string        `json:"prev_hash"`
	Hash         string        `json:"hash"`
}

// NewBlock creates a new block linked to the previous block's hash.
func NewBlock(index uint64, transactions []Transaction, validator string, prevHash string) Block {
	b := Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		Validator:    validator,
		PrevHash:     prevHash,
	}
	b.Hash = b.CalculateHash()
	return b
}

// CalculateHash computes the SHA-256 hash of the block's contents.
// Validator is included so a different proposer produces a different hash.
func (b Block) CalculateHash() string {
	txData, _ := json.Marshal(b.Transactions)
	record := fmt.Sprintf("%d%d%s%s%s", b.Index, b.Timestamp, string(txData), b.Validator, b.PrevHash)
	hash := sha256.Sum256([]byte(record))
	return fmt.Sprintf("%x", hash)
}
