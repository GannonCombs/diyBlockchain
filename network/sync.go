package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gannoncombs/diyBlockchain/core"
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

// syncWithPeers asks all known peers for their chain and adopts the longest valid one.
func (n *Node) syncWithPeers() {
	n.mu.Lock()
	peers := make([]string, 0, len(n.peers))
	for p := range n.peers {
		peers = append(peers, p)
	}
	ourHeight := len(n.store.Chain().Blocks)
	n.mu.Unlock()

	var bestBlocks []core.Block
	bestHeight := ourHeight

	for _, peer := range peers {
		blocks, err := fetchChain(peer)
		if err != nil {
			log.Printf("[node:%s] Failed to sync from %s: %s", n.port, peer, err)
			continue
		}

		if len(blocks) > bestHeight {
			bestBlocks = blocks
			bestHeight = len(blocks)
		}
	}

	if bestBlocks != nil {
		n.mu.Lock()
		defer n.mu.Unlock()

		if err := n.store.ReplaceChain(bestBlocks); err != nil {
			log.Printf("[node:%s] Chain replacement failed: %s", n.port, err)
		} else {
			log.Printf("[node:%s] Synced chain: %d -> %d blocks", n.port, ourHeight, bestHeight)
		}
	}
}

// broadcastBlock sends a block to all known peers.
func (n *Node) broadcastBlock(block core.Block) {
	n.mu.Lock()
	peers := make([]string, 0, len(n.peers))
	for p := range n.peers {
		peers = append(peers, p)
	}
	n.mu.Unlock()

	data, _ := json.Marshal(block)

	for _, peer := range peers {
		resp, err := httpClient.Post(peer+"/block", "application/json", bytes.NewReader(data))
		if err != nil {
			log.Printf("[node:%s] Failed to send block to %s: %s", n.port, peer, err)
			continue
		}
		resp.Body.Close()
	}
}

// broadcastTx sends a transaction to all known peers.
func (n *Node) broadcastTx(tx core.Transaction) {
	n.mu.Lock()
	peers := make([]string, 0, len(n.peers))
	for p := range n.peers {
		peers = append(peers, p)
	}
	n.mu.Unlock()

	data, _ := json.Marshal(tx)

	for _, peer := range peers {
		resp, err := httpClient.Post(peer+"/tx", "application/json", bytes.NewReader(data))
		if err != nil {
			continue
		}
		resp.Body.Close()
	}
}

// RegisterWithPeer tells a peer about our existence.
func RegisterWithPeer(peerURL, ourURL string) error {
	body := fmt.Sprintf(`{"url":"%s"}`, ourURL)
	resp, err := httpClient.Post(peerURL+"/peers", "application/json",
		bytes.NewReader([]byte(body)))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// fetchChain downloads the full chain from a peer.
func fetchChain(peerURL string) ([]core.Block, error) {
	resp, err := httpClient.Get(peerURL + "/chain")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var blocks []core.Block
	if err := json.NewDecoder(resp.Body).Decode(&blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// FetchPeers downloads the peer list from a peer.
func FetchPeers(peerURL string) ([]string, error) {
	resp, err := httpClient.Get(peerURL + "/peers")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var peers []string
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, err
	}
	return peers, nil
}
