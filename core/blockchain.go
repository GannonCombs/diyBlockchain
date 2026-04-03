package core

import "fmt"

// Blockchain is an ordered list of blocks, starting from the genesis block.
type Blockchain struct {
	Blocks []Block
}

// NewBlockchain creates a new blockchain with the genesis block already in place.
func NewBlockchain() *Blockchain {
	genesis := newGenesisBlock()
	return &Blockchain{
		Blocks: []Block{genesis},
	}
}

func newGenesisBlock() Block {
	b := Block{
		Index:        0,
		Timestamp:    1743552000,
		Transactions: []Transaction{},
		Validator:    "",
		PrevHash:     "",
	}
	b.Hash = b.CalculateHash()
	return b
}

// AddBlock appends a new block proposed by the given validator.
func (bc *Blockchain) AddBlock(transactions []Transaction, validator string) Block {
	prev := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(prev.Index+1, transactions, validator, prev.Hash)
	bc.Blocks = append(bc.Blocks, newBlock)
	return newBlock
}

// LatestBlock returns the most recent block in the chain.
func (bc *Blockchain) LatestBlock() Block {
	return bc.Blocks[len(bc.Blocks)-1]
}

// IsValid checks the integrity of the entire chain.
func (bc *Blockchain) IsValid() error {
	for i := 1; i < len(bc.Blocks); i++ {
		current := bc.Blocks[i]
		prev := bc.Blocks[i-1]

		if current.Hash != current.CalculateHash() {
			return fmt.Errorf("block %d has been tampered with: stored hash doesn't match computed hash", i)
		}

		if current.PrevHash != prev.Hash {
			return fmt.Errorf("block %d has broken chain link: PrevHash doesn't match previous block's Hash", i)
		}
	}

	if bc.Blocks[0].Hash != bc.Blocks[0].CalculateHash() {
		return fmt.Errorf("genesis block has been tampered with")
	}

	return nil
}
