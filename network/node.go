package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gannoncombs/diyBlockchain/consensus"
	"github.com/gannoncombs/diyBlockchain/core"
	"github.com/gannoncombs/diyBlockchain/persistence"
)

const blockInterval = 10 * time.Second

// Node is a blockchain node that serves an HTTP API, produces blocks,
// and communicates with peers.
type Node struct {
	address  string // This node's validator address (wallet)
	port     string
	store    *persistence.Store
	genesis  core.Genesis
	peers    map[string]bool // Known peer URLs (e.g., "http://localhost:3001")
	mempool  []core.Transaction
	seenTxs  map[string]bool // Dedup: prevents relaying the same tx in circles
	mu       sync.Mutex
	stopCh   chan struct{}
}

// NewNode creates a node but doesn't start it yet.
func NewNode(port string, address string, store *persistence.Store, genesis core.Genesis) *Node {
	return &Node{
		address: address,
		port:    port,
		store:   store,
		genesis: genesis,
		peers:   make(map[string]bool),
		seenTxs: make(map[string]bool),
		stopCh:  make(chan struct{}),
	}
}

// AddPeer registers a peer URL.
func (n *Node) AddPeer(url string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	self := "http://localhost:" + n.port
	if url != self {
		n.peers[url] = true
	}
}

// Start launches the HTTP server and block production loop.
func (n *Node) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", n.handleStatus)
	mux.HandleFunc("/chain", n.handleChain)
	mux.HandleFunc("/block", n.handleBlock)
	mux.HandleFunc("/tx", n.handleTx)
	mux.HandleFunc("/peers", n.handlePeers)

	server := &http.Server{
		Addr:    ":" + n.port,
		Handler: mux,
	}

	// Sync with peers before starting
	n.syncWithPeers()

	// Start block production in background
	go n.produceBlocks()

	log.Printf("[node:%s] Listening on port %s (validator: %s)", n.port, n.port, n.shortAddr())
	return server.ListenAndServe()
}

// Stop signals the block production loop to stop.
func (n *Node) Stop() {
	close(n.stopCh)
}

func (n *Node) shortAddr() string {
	if len(n.address) > 12 {
		return n.address[:12] + "..."
	}
	return n.address
}

// --- HTTP Handlers ---

func (n *Node) handleStatus(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()

	bc := n.store.Chain()
	state := n.store.State()

	validatorCount := 0
	for _, stake := range state.StakedBalances {
		if stake >= consensus.MinStake {
			validatorCount++
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"height":      len(bc.Blocks),
		"latest_hash": bc.LatestBlock().Hash,
		"validators":  validatorCount,
		"peers":       len(n.peers),
		"mempool":     len(n.mempool),
		"address":     n.address,
	})
}

func (n *Node) handleChain(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()
	json.NewEncoder(w).Encode(n.store.Chain().Blocks)
}

func (n *Node) handleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var block core.Block
	if err := json.NewDecoder(r.Body).Decode(&block); err != nil {
		http.Error(w, "invalid block: "+err.Error(), http.StatusBadRequest)
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// If this block is ahead of us by more than 1, we need to sync
	latest := n.store.Chain().LatestBlock()
	if block.Index > latest.Index+1 {
		log.Printf("[node:%s] Block %d is ahead of us (at %d), syncing...", n.port, block.Index, latest.Index)
		go n.syncWithPeers()
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Try to accept the block
	if err := n.store.AcceptBlock(block); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Remove any mempool transactions that are now in this block
	n.pruneMempool(block.Transactions)

	valDisplay := block.Validator
	if len(valDisplay) > 12 {
		valDisplay = valDisplay[:12] + "..."
	}
	log.Printf("[node:%s] Accepted block %d from peer (validator: %s)",
		n.port, block.Index, valDisplay)

	w.WriteHeader(http.StatusOK)
}

func (n *Node) handleTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var tx core.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, "invalid transaction: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := tx.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	n.mu.Lock()
	key := txKey(tx)
	if n.seenTxs[key] {
		n.mu.Unlock()
		w.WriteHeader(http.StatusOK) // Already have it — don't relay again
		return
	}
	n.seenTxs[key] = true
	n.mempool = append(n.mempool, tx)
	n.mu.Unlock()

	log.Printf("[node:%s] Added tx to mempool (%s -> %s: %d)", n.port, tx.From, tx.To, tx.Amount)

	// Gossip to peers
	go n.broadcastTx(tx)

	w.WriteHeader(http.StatusOK)
}

func (n *Node) handlePeers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		n.mu.Lock()
		peers := make([]string, 0, len(n.peers))
		for p := range n.peers {
			peers = append(peers, p)
		}
		n.mu.Unlock()
		json.NewEncoder(w).Encode(peers)

	case http.MethodPost:
		var body struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
			http.Error(w, "invalid peer URL", http.StatusBadRequest)
			return
		}
		n.AddPeer(body.URL)
		log.Printf("[node:%s] Registered peer: %s", n.port, body.URL)
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
	}
}

// --- Block Production ---

func (n *Node) produceBlocks() {
	ticker := time.NewTicker(blockInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			n.tryProduceBlock()
		case <-n.stopCh:
			return
		}
	}
}

func (n *Node) tryProduceBlock() {
	n.mu.Lock()
	defer n.mu.Unlock()

	state := n.store.State()
	seed := n.store.Chain().LatestBlock().Hash

	validator, err := consensus.SelectValidator(state, seed)
	if err != nil {
		return // No validators — can't produce
	}

	// Only produce if WE are the selected validator
	if validator != n.address {
		return
	}

	// Build transaction list: reward + mempool
	txs := []core.Transaction{consensus.CreateBlockReward(n.address)}
	txs = append(txs, n.mempool...)

	block, err := n.store.AddBlock(txs, n.address)
	if err != nil {
		log.Printf("[node:%s] Failed to produce block: %s", n.port, err)
		return
	}

	n.mempool = nil                    // Clear mempool
	n.seenTxs = make(map[string]bool) // Reset dedup set

	log.Printf("[node:%s] Produced block %d (hash: %s)",
		n.port, block.Index, block.Hash[:16]+"...")

	// Broadcast to peers
	go n.broadcastBlock(block)
}

// txKey returns a unique string for a transaction, used for deduplication.
func txKey(tx core.Transaction) string {
	return fmt.Sprintf("%s%s%s%d%d", tx.Type, tx.From, tx.To, tx.Amount, tx.Timestamp)
}

func (n *Node) pruneMempool(blockTxs []core.Transaction) {
	// Build a set of transaction signatures/hashes in the block for fast lookup
	inBlock := make(map[string]bool)
	for _, tx := range blockTxs {
		key := fmt.Sprintf("%s%s%s%d%d", tx.Type, tx.From, tx.To, tx.Amount, tx.Timestamp)
		inBlock[key] = true
	}

	var remaining []core.Transaction
	for _, tx := range n.mempool {
		key := fmt.Sprintf("%s%s%s%d%d", tx.Type, tx.From, tx.To, tx.Amount, tx.Timestamp)
		if !inBlock[key] {
			remaining = append(remaining, tx)
		}
	}
	n.mempool = remaining
}
