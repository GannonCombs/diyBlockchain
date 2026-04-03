package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gannoncombs/diyBlockchain/consensus"
	"github.com/gannoncombs/diyBlockchain/core"
	"github.com/gannoncombs/diyBlockchain/persistence"
)

func tempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "node-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// bootstrapStore creates a store with validators set up.
func bootstrapStore(t *testing.T, validators map[string]uint64) *persistence.Store {
	dir := tempDir(t)
	genesis := core.DefaultGenesis()
	store, err := persistence.NewStore(dir, genesis)
	if err != nil {
		t.Fatal(err)
	}

	// Distribute and stake
	var distTxs []core.Transaction
	var stakeTxs []core.Transaction
	for addr, amount := range validators {
		distTxs = append(distTxs, core.NewTransaction("treasury", addr, amount))
		stakeTxs = append(stakeTxs, core.NewStakeTransaction(addr, amount/2))
	}

	store.AddBlock(distTxs, "")
	store.AddBlock(stakeTxs, "")
	return store
}

func TestNodeStatus(t *testing.T) {
	store := bootstrapStore(t, map[string]uint64{"alice": 10000})
	node := NewNode("9100", "alice", store, core.DefaultGenesis())

	go node.Start()
	time.Sleep(200 * time.Millisecond)
	defer node.Stop()

	resp, err := http.Get("http://localhost:9100/status")
	if err != nil {
		t.Fatalf("status request failed: %s", err)
	}
	defer resp.Body.Close()

	var status map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&status)

	if status["address"] != "alice" {
		t.Errorf("expected address alice, got %s", status["address"])
	}
	if int(status["height"].(float64)) != 3 {
		t.Errorf("expected height 3, got %v", status["height"])
	}
}

func TestNodeReceivesBlock(t *testing.T) {
	store := bootstrapStore(t, map[string]uint64{"alice": 10000})
	node := NewNode("9101", "alice", store, core.DefaultGenesis())

	go node.Start()
	time.Sleep(200 * time.Millisecond)
	defer node.Stop()

	// Produce a block externally and send it
	state := store.State()
	seed := store.Chain().LatestBlock().Hash
	validator, _ := consensus.SelectValidator(state, seed)

	txs := []core.Transaction{consensus.CreateBlockReward(validator)}
	prev := store.Chain().LatestBlock()
	block := core.NewBlock(prev.Index+1, txs, validator, prev.Hash)

	data, _ := json.Marshal(block)
	resp, err := http.Post("http://localhost:9101/block", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("block POST failed: %s", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Verify chain grew
	if len(store.Chain().Blocks) != 4 {
		t.Errorf("expected 4 blocks, got %d", len(store.Chain().Blocks))
	}
}

func TestSubmitTransaction(t *testing.T) {
	store := bootstrapStore(t, map[string]uint64{"alice": 10000})
	node := NewNode("9102", "alice", store, core.DefaultGenesis())

	go node.Start()
	time.Sleep(200 * time.Millisecond)
	defer node.Stop()

	tx := core.NewTransaction("alice", "bob", 100)
	data, _ := json.Marshal(tx)

	resp, err := http.Post("http://localhost:9102/tx", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("tx POST failed: %s", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestPeerRegistration(t *testing.T) {
	store := bootstrapStore(t, map[string]uint64{"alice": 10000})
	node := NewNode("9103", "alice", store, core.DefaultGenesis())

	go node.Start()
	time.Sleep(200 * time.Millisecond)
	defer node.Stop()

	// Register a peer
	body := `{"url":"http://localhost:9999"}`
	resp, err := http.Post("http://localhost:9103/peers", "application/json",
		bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Get peers
	resp, err = http.Get("http://localhost:9103/peers")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var peers []string
	json.NewDecoder(resp.Body).Decode(&peers)

	if len(peers) != 1 || peers[0] != "http://localhost:9999" {
		t.Errorf("expected [http://localhost:9999], got %v", peers)
	}
}

func TestChainSync(t *testing.T) {
	// Node 1: has extra blocks
	store1 := bootstrapStore(t, map[string]uint64{"alice": 10000, "bob": 5000})
	state := store1.State()
	seed := store1.Chain().LatestBlock().Hash
	validator, _ := consensus.SelectValidator(state, seed)
	store1.AddBlock([]core.Transaction{consensus.CreateBlockReward(validator)}, validator)

	node1 := NewNode("9104", "alice", store1, core.DefaultGenesis())
	go node1.Start()
	time.Sleep(200 * time.Millisecond)
	defer node1.Stop()

	// Node 2: fresh bootstrap (shorter chain)
	store2 := bootstrapStore(t, map[string]uint64{"alice": 10000, "bob": 5000})
	node2 := NewNode("9105", "bob", store2, core.DefaultGenesis())
	node2.AddPeer("http://localhost:9104")

	go node2.Start()
	time.Sleep(500 * time.Millisecond) // Give it time to sync
	defer node2.Stop()

	height1 := len(store1.Chain().Blocks)
	height2 := len(store2.Chain().Blocks)

	if height2 != height1 {
		t.Errorf("node2 should have synced to %d blocks, got %d", height1, height2)
	}

	fmt.Printf("Node1: %d blocks, Node2: %d blocks (synced)\n", height1, height2)
}
