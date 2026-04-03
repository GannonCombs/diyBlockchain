package core

import (
	"fmt"
	"time"
)

// Transaction types
const (
	TxTransfer = "transfer" // Regular token transfer
	TxStake    = "stake"    // Lock tokens as validator collateral
	TxUnstake  = "unstake"  // Unlock staked tokens
)

// Transaction represents a state change in the blockchain.
// The Type field determines how it's processed:
//   - "transfer": move tokens from one account to another
//   - "stake":    lock tokens (From's spendable balance -> From's staked balance)
//   - "unstake":  unlock tokens (From's staked balance -> From's spendable balance)
type Transaction struct {
	Type      string `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature,omitempty"` // Hex-encoded ECDSA signature
	PubKey    string `json:"pub_key,omitempty"`   // Hex-encoded sender's public key
}

// NewTransaction creates a regular transfer transaction.
func NewTransaction(from, to string, amount uint64) Transaction {
	return Transaction{
		Type:      TxTransfer,
		From:      from,
		To:        to,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}
}

// NewStakeTransaction creates a staking transaction.
// "To" is unused — the tokens stay with the sender, just locked.
func NewStakeTransaction(from string, amount uint64) Transaction {
	return Transaction{
		Type:      TxStake,
		From:      from,
		To:        from,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}
}

// NewUnstakeTransaction creates an unstaking transaction.
func NewUnstakeTransaction(from string, amount uint64) Transaction {
	return Transaction{
		Type:      TxUnstake,
		From:      from,
		To:        from,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}
}

// IsCoinbase returns true if this is a reward transaction (no sender).
func (tx Transaction) IsCoinbase() bool {
	return tx.From == ""
}

// Validate checks whether a transaction is well-formed.
func (tx Transaction) Validate() error {
	if tx.Amount == 0 {
		return fmt.Errorf("transaction amount must be greater than 0")
	}
	if tx.To == "" {
		return fmt.Errorf("transaction must have a recipient")
	}

	switch tx.Type {
	case TxTransfer:
		if !tx.IsCoinbase() && tx.From == tx.To {
			return fmt.Errorf("sender and receiver must be different")
		}
	case TxStake, TxUnstake:
		if tx.From == "" {
			return fmt.Errorf("%s transaction must have a sender", tx.Type)
		}
	default:
		return fmt.Errorf("unknown transaction type: %s", tx.Type)
	}

	return nil
}
