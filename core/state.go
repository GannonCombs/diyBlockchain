package core

import "fmt"

// State tracks both spendable and staked balances for all accounts.
// Staked tokens can't be spent — they're locked as validator collateral.
type State struct {
	Balances       map[string]uint64 // Spendable tokens
	StakedBalances map[string]uint64 // Locked tokens (used for validator selection)
}

// NewStateFromChain rebuilds the current state by replaying the entire blockchain.
func NewStateFromChain(genesis Genesis, bc *Blockchain) (*State, error) {
	s := &State{
		Balances:       make(map[string]uint64),
		StakedBalances: make(map[string]uint64),
	}

	// Start with genesis balances
	for account, balance := range genesis.Balances {
		s.Balances[account] = balance
	}

	// Replay every transaction in every block
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if err := s.Apply(tx); err != nil {
				return nil, fmt.Errorf("invalid tx in block %d: %w", block.Index, err)
			}
		}
	}

	return s, nil
}

// Apply executes a single transaction against the current state.
func (s *State) Apply(tx Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}

	switch tx.Type {
	case TxTransfer:
		return s.applyTransfer(tx)
	case TxStake:
		return s.applyStake(tx)
	case TxUnstake:
		return s.applyUnstake(tx)
	default:
		return fmt.Errorf("unknown transaction type: %s", tx.Type)
	}
}

func (s *State) applyTransfer(tx Transaction) error {
	if tx.IsCoinbase() {
		s.Balances[tx.To] += tx.Amount
		return nil
	}

	if s.Balances[tx.From] < tx.Amount {
		return fmt.Errorf("%s has %d tokens but tried to send %d",
			tx.From, s.Balances[tx.From], tx.Amount)
	}

	s.Balances[tx.From] -= tx.Amount
	s.Balances[tx.To] += tx.Amount
	return nil
}

// applyStake moves tokens from spendable to staked.
func (s *State) applyStake(tx Transaction) error {
	if s.Balances[tx.From] < tx.Amount {
		return fmt.Errorf("%s has %d spendable tokens but tried to stake %d",
			tx.From, s.Balances[tx.From], tx.Amount)
	}

	s.Balances[tx.From] -= tx.Amount
	s.StakedBalances[tx.From] += tx.Amount
	return nil
}

// applyUnstake moves tokens from staked back to spendable.
func (s *State) applyUnstake(tx Transaction) error {
	if s.StakedBalances[tx.From] < tx.Amount {
		return fmt.Errorf("%s has %d staked tokens but tried to unstake %d",
			tx.From, s.StakedBalances[tx.From], tx.Amount)
	}

	s.StakedBalances[tx.From] -= tx.Amount
	s.Balances[tx.From] += tx.Amount
	return nil
}
