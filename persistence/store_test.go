package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gannoncombs/diyBlockchain/core"
)

func tempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "blockchain-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestNewStoreCreatesGenesis(t *testing.T) {
	dir := tempDir(t)
	store, err := NewStore(dir, core.DefaultGenesis())
	if err != nil {
		t.Fatalf("NewStore failed: %s", err)
	}
	defer store.Close()

	if len(store.Chain().Blocks) != 1 {
		t.Errorf("expected 1 block (genesis), got %d", len(store.Chain().Blocks))
	}

	if store.State().Balances["treasury"] != 1000000 {
		t.Errorf("expected treasury balance 1000000, got %d", store.State().Balances["treasury"])
	}

	// File should exist
	if _, err := os.Stat(filepath.Join(dir, chainFileName)); os.IsNotExist(err) {
		t.Error("chain file should exist on disk")
	}
}

func TestAddBlockPersists(t *testing.T) {
	dir := tempDir(t)
	store, err := NewStore(dir, core.DefaultGenesis())
	if err != nil {
		t.Fatalf("NewStore failed: %s", err)
	}

	_, err = store.AddBlock([]core.Transaction{
		core.NewTransaction("treasury", "alice", 500),
	}, "val1")
	if err != nil {
		t.Fatalf("AddBlock failed: %s", err)
	}
	store.Close()

	// Reopen and verify the block survived
	store2, err := NewStore(dir, core.DefaultGenesis())
	if err != nil {
		t.Fatalf("reload failed: %s", err)
	}
	defer store2.Close()

	if len(store2.Chain().Blocks) != 2 {
		t.Errorf("expected 2 blocks after reload, got %d", len(store2.Chain().Blocks))
	}

	if store2.State().Balances["alice"] != 500 {
		t.Errorf("expected alice balance 500 after reload, got %d", store2.State().Balances["alice"])
	}
}

func TestRoundTripMultipleBlocks(t *testing.T) {
	dir := tempDir(t)
	store, err := NewStore(dir, core.DefaultGenesis())
	if err != nil {
		t.Fatal(err)
	}

	store.AddBlock([]core.Transaction{
		core.NewTransaction("treasury", "alice", 5000),
		core.NewTransaction("treasury", "bob", 3000),
	}, "v1")
	store.AddBlock([]core.Transaction{
		core.NewStakeTransaction("alice", 2000),
	}, "v1")
	store.AddBlock([]core.Transaction{
		core.NewTransaction("alice", "bob", 100),
	}, "v1")
	store.Close()

	// Reload
	store2, err := NewStore(dir, core.DefaultGenesis())
	if err != nil {
		t.Fatalf("reload failed: %s", err)
	}
	defer store2.Close()

	if len(store2.Chain().Blocks) != 4 {
		t.Fatalf("expected 4 blocks, got %d", len(store2.Chain().Blocks))
	}

	state := store2.State()
	// alice: 5000 - 2000 (staked) - 100 = 2900
	if state.Balances["alice"] != 2900 {
		t.Errorf("alice spendable should be 2900, got %d", state.Balances["alice"])
	}
	if state.StakedBalances["alice"] != 2000 {
		t.Errorf("alice staked should be 2000, got %d", state.StakedBalances["alice"])
	}
}

func TestRejectsInvalidTransaction(t *testing.T) {
	dir := tempDir(t)
	store, err := NewStore(dir, core.DefaultGenesis())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// alice has no tokens — this should fail
	_, err = store.AddBlock([]core.Transaction{
		core.NewTransaction("alice", "bob", 500),
	}, "v1")
	if err == nil {
		t.Error("should reject block with invalid transaction")
	}

	// Chain should still be just genesis
	if len(store.Chain().Blocks) != 1 {
		t.Errorf("chain should still have 1 block, got %d", len(store.Chain().Blocks))
	}
}

func TestDetectsTamperedFile(t *testing.T) {
	dir := tempDir(t)
	store, err := NewStore(dir, core.DefaultGenesis())
	if err != nil {
		t.Fatal(err)
	}
	store.AddBlock([]core.Transaction{
		core.NewTransaction("treasury", "alice", 100),
	}, "v1")
	store.Close()

	// Corrupt the file by modifying a byte
	chainPath := filepath.Join(dir, chainFileName)
	data, _ := os.ReadFile(chainPath)
	// Find "alice" and change it to "zlice"
	for i := range data {
		if i+5 <= len(data) && string(data[i:i+5]) == "alice" {
			data[i] = 'z'
			break
		}
	}
	os.WriteFile(chainPath, data, 0644)

	// Reload should fail validation
	_, err = NewStore(dir, core.DefaultGenesis())
	if err == nil {
		t.Error("should detect tampered chain file")
	}
}

func TestLoadGenesisFromFile(t *testing.T) {
	dir := tempDir(t)

	// Write a custom genesis file
	genesisPath := filepath.Join(dir, "genesis.json")
	os.WriteFile(genesisPath, []byte(`{"balances":{"founder":500000}}`), 0644)

	genesis, err := core.LoadGenesis(genesisPath)
	if err != nil {
		t.Fatalf("LoadGenesis failed: %s", err)
	}

	if genesis.Balances["founder"] != 500000 {
		t.Errorf("expected founder balance 500000, got %d", genesis.Balances["founder"])
	}

	// Use it with a store
	store, err := NewStore(dir, genesis)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if store.State().Balances["founder"] != 500000 {
		t.Errorf("store should use custom genesis")
	}
}
