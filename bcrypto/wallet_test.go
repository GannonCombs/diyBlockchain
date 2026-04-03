package bcrypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gannoncombs/diyBlockchain/core"
)

func tempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "wallet-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// --- Wallet Tests ---

func TestNewWalletGeneratesUniqueAddress(t *testing.T) {
	w1, err := NewWallet()
	if err != nil {
		t.Fatalf("NewWallet failed: %s", err)
	}
	w2, err := NewWallet()
	if err != nil {
		t.Fatalf("NewWallet failed: %s", err)
	}

	if w1.Address == w2.Address {
		t.Error("two wallets should have different addresses")
	}

	// Address should be 40 hex characters (20 bytes)
	if len(w1.Address) != 40 {
		t.Errorf("address should be 40 chars, got %d", len(w1.Address))
	}
}

func TestWalletSaveAndLoad(t *testing.T) {
	dir := tempDir(t)
	w1, _ := NewWallet()
	w1.Save(dir)

	path := filepath.Join(dir, w1.Address+".json")
	w2, err := LoadWallet(path)
	if err != nil {
		t.Fatalf("LoadWallet failed: %s", err)
	}

	if w1.Address != w2.Address {
		t.Errorf("address mismatch after reload: %s vs %s", w1.Address, w2.Address)
	}

	// Verify the private key survived the round trip by signing with both
	tx := core.NewTransaction(w1.Address, "someone", 100)
	err = SignTransaction(&tx, w1.PrivateKey)
	if err != nil {
		t.Fatalf("signing with original wallet failed: %s", err)
	}
	if err := VerifyTransaction(tx); err != nil {
		t.Errorf("signature from original wallet should verify: %s", err)
	}

	tx2 := core.NewTransaction(w2.Address, "someone", 100)
	err = SignTransaction(&tx2, w2.PrivateKey)
	if err != nil {
		t.Fatalf("signing with reloaded wallet failed: %s", err)
	}
	if err := VerifyTransaction(tx2); err != nil {
		t.Errorf("signature from reloaded wallet should verify: %s", err)
	}
}

func TestListWallets(t *testing.T) {
	dir := tempDir(t)

	w1, _ := NewWallet()
	w2, _ := NewWallet()
	w1.Save(dir)
	w2.Save(dir)

	addrs, err := ListWallets(dir)
	if err != nil {
		t.Fatalf("ListWallets failed: %s", err)
	}

	if len(addrs) != 2 {
		t.Errorf("expected 2 wallets, got %d", len(addrs))
	}
}

// --- Signature Tests ---

func TestSignAndVerify(t *testing.T) {
	w, _ := NewWallet()
	tx := core.NewTransaction(w.Address, "recipient", 50)

	if err := SignTransaction(&tx, w.PrivateKey); err != nil {
		t.Fatalf("signing failed: %s", err)
	}

	if err := VerifyTransaction(tx); err != nil {
		t.Errorf("valid signature should verify: %s", err)
	}
}

func TestRejectUnsigned(t *testing.T) {
	tx := core.NewTransaction("someaddr", "recipient", 50)
	if err := VerifyTransaction(tx); err == nil {
		t.Error("unsigned transaction should be rejected")
	}
}

func TestRejectTamperedAmount(t *testing.T) {
	w, _ := NewWallet()
	tx := core.NewTransaction(w.Address, "recipient", 50)
	SignTransaction(&tx, w.PrivateKey)

	// Tamper with the amount after signing
	tx.Amount = 999999

	if err := VerifyTransaction(tx); err == nil {
		t.Error("tampered transaction should be rejected")
	}
}

func TestRejectWrongSigner(t *testing.T) {
	w1, _ := NewWallet()
	w2, _ := NewWallet()

	// Create tx from w1's address but sign with w2's key
	tx := core.NewTransaction(w1.Address, "recipient", 50)
	SignTransaction(&tx, w2.PrivateKey)

	if err := VerifyTransaction(tx); err == nil {
		t.Error("transaction signed by wrong key should be rejected")
	}
}

func TestCoinbaseSkipsVerification(t *testing.T) {
	tx := core.NewTransaction("", "miner", 100) // coinbase
	if err := VerifyTransaction(tx); err != nil {
		t.Errorf("coinbase should skip verification: %s", err)
	}
}

func TestSignStakeTransaction(t *testing.T) {
	w, _ := NewWallet()
	tx := core.NewStakeTransaction(w.Address, 1000)
	SignTransaction(&tx, w.PrivateKey)

	if err := VerifyTransaction(tx); err != nil {
		t.Errorf("signed stake tx should verify: %s", err)
	}
}
