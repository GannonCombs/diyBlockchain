package core

import "testing"

// --- Block & Chain Tests ---

func TestGenesisBlock(t *testing.T) {
	bc := NewBlockchain()

	if len(bc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(bc.Blocks))
	}

	genesis := bc.Blocks[0]
	if genesis.Index != 0 {
		t.Errorf("genesis index should be 0, got %d", genesis.Index)
	}
	if genesis.PrevHash != "" {
		t.Errorf("genesis PrevHash should be empty, got %s", genesis.PrevHash)
	}
	if genesis.Hash == "" {
		t.Error("genesis hash should not be empty")
	}
}

func TestAddBlock(t *testing.T) {
	bc := NewBlockchain()
	txs := []Transaction{NewTransaction("alice", "bob", 50)}
	bc.AddBlock(txs, "validator1")
	bc.AddBlock([]Transaction{NewTransaction("bob", "charlie", 20)}, "validator1")

	if len(bc.Blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(bc.Blocks))
	}

	for i := 1; i < len(bc.Blocks); i++ {
		if bc.Blocks[i].PrevHash != bc.Blocks[i-1].Hash {
			t.Errorf("block %d PrevHash doesn't match block %d Hash", i, i-1)
		}
	}
}

func TestBlockIncludesValidator(t *testing.T) {
	bc := NewBlockchain()
	block := bc.AddBlock([]Transaction{NewTransaction("a", "b", 10)}, "val1")

	if block.Validator != "val1" {
		t.Errorf("expected validator val1, got %s", block.Validator)
	}
}

func TestIsValid(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock([]Transaction{NewTransaction("alice", "bob", 50)}, "v")
	bc.AddBlock([]Transaction{NewTransaction("bob", "charlie", 20)}, "v")

	if err := bc.IsValid(); err != nil {
		t.Errorf("valid chain should pass validation, got: %s", err)
	}
}

func TestDetectsTampering(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock([]Transaction{NewTransaction("alice", "bob", 50)}, "v")
	bc.AddBlock([]Transaction{NewTransaction("bob", "charlie", 20)}, "v")

	bc.Blocks[1].Transactions[0].Amount = 999999

	if err := bc.IsValid(); err == nil {
		t.Error("chain with tampered transaction should fail validation")
	}
}

// --- Transaction Validation Tests ---

func TestValidTransaction(t *testing.T) {
	tx := NewTransaction("alice", "bob", 50)
	if err := tx.Validate(); err != nil {
		t.Errorf("valid transaction should pass: %s", err)
	}
}

func TestZeroAmountRejected(t *testing.T) {
	tx := NewTransaction("alice", "bob", 0)
	if err := tx.Validate(); err == nil {
		t.Error("zero amount should be rejected")
	}
}

func TestSelfTransferRejected(t *testing.T) {
	tx := NewTransaction("alice", "alice", 50)
	if err := tx.Validate(); err == nil {
		t.Error("self-transfer should be rejected")
	}
}

func TestCoinbaseTransaction(t *testing.T) {
	tx := NewTransaction("", "alice", 100)
	if !tx.IsCoinbase() {
		t.Error("transaction with empty From should be coinbase")
	}
	if err := tx.Validate(); err != nil {
		t.Errorf("coinbase transaction should be valid: %s", err)
	}
}

func TestStakeTransaction(t *testing.T) {
	tx := NewStakeTransaction("alice", 1000)
	if err := tx.Validate(); err != nil {
		t.Errorf("valid stake tx should pass: %s", err)
	}
	if tx.Type != TxStake {
		t.Errorf("expected type %s, got %s", TxStake, tx.Type)
	}
}

func TestUnstakeTransaction(t *testing.T) {
	tx := NewUnstakeTransaction("alice", 500)
	if err := tx.Validate(); err != nil {
		t.Errorf("valid unstake tx should pass: %s", err)
	}
	if tx.Type != TxUnstake {
		t.Errorf("expected type %s, got %s", TxUnstake, tx.Type)
	}
}

// --- State Tests ---

func TestGenesisState(t *testing.T) {
	bc := NewBlockchain()
	state, err := NewStateFromChain(DefaultGenesis(), bc)
	if err != nil {
		t.Fatalf("failed to build state: %s", err)
	}
	if state.Balances["treasury"] != 1000000 {
		t.Errorf("treasury should have 1000000, got %d", state.Balances["treasury"])
	}
}

func TestStateAfterTransactions(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock([]Transaction{
		NewTransaction("treasury", "alice", 500),
		NewTransaction("treasury", "bob", 300),
	}, "v")
	bc.AddBlock([]Transaction{
		NewTransaction("alice", "bob", 100),
	}, "v")

	state, err := NewStateFromChain(DefaultGenesis(), bc)
	if err != nil {
		t.Fatalf("failed to build state: %s", err)
	}

	if state.Balances["treasury"] != 999200 {
		t.Errorf("treasury should have 999200, got %d", state.Balances["treasury"])
	}
	if state.Balances["alice"] != 400 {
		t.Errorf("alice should have 400, got %d", state.Balances["alice"])
	}
	if state.Balances["bob"] != 400 {
		t.Errorf("bob should have 400, got %d", state.Balances["bob"])
	}
}

func TestOverspendRejected(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock([]Transaction{
		NewTransaction("treasury", "alice", 100),
	}, "v")
	bc.AddBlock([]Transaction{
		NewTransaction("alice", "bob", 200),
	}, "v")

	_, err := NewStateFromChain(DefaultGenesis(), bc)
	if err == nil {
		t.Error("overspend should be rejected")
	}
}

func TestStakingMovesBalance(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock([]Transaction{
		NewTransaction("treasury", "alice", 5000),
	}, "v")
	bc.AddBlock([]Transaction{
		NewStakeTransaction("alice", 3000),
	}, "v")

	state, err := NewStateFromChain(DefaultGenesis(), bc)
	if err != nil {
		t.Fatalf("failed to build state: %s", err)
	}

	if state.Balances["alice"] != 2000 {
		t.Errorf("alice spendable should be 2000, got %d", state.Balances["alice"])
	}
	if state.StakedBalances["alice"] != 3000 {
		t.Errorf("alice staked should be 3000, got %d", state.StakedBalances["alice"])
	}
}

func TestUnstakingReturnsBalance(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock([]Transaction{
		NewTransaction("treasury", "alice", 5000),
	}, "v")
	bc.AddBlock([]Transaction{
		NewStakeTransaction("alice", 3000),
	}, "v")
	bc.AddBlock([]Transaction{
		NewUnstakeTransaction("alice", 1000),
	}, "v")

	state, err := NewStateFromChain(DefaultGenesis(), bc)
	if err != nil {
		t.Fatalf("failed to build state: %s", err)
	}

	if state.Balances["alice"] != 3000 {
		t.Errorf("alice spendable should be 3000, got %d", state.Balances["alice"])
	}
	if state.StakedBalances["alice"] != 2000 {
		t.Errorf("alice staked should be 2000, got %d", state.StakedBalances["alice"])
	}
}

func TestCannotOverstake(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock([]Transaction{
		NewTransaction("treasury", "alice", 1000),
	}, "v")
	bc.AddBlock([]Transaction{
		NewStakeTransaction("alice", 2000), // Only has 1000
	}, "v")

	_, err := NewStateFromChain(DefaultGenesis(), bc)
	if err == nil {
		t.Error("staking more than spendable balance should be rejected")
	}
}
