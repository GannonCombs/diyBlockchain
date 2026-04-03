package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/gannoncombs/diyBlockchain/core"
)

const (
	// MinStake is the minimum tokens required to become a validator.
	MinStake uint64 = 1000

	// BlockReward is how many new tokens the selected validator earns per block.
	BlockReward uint64 = 100
)

// SelectValidator picks who gets to propose the next block.
// Selection is weighted by stake — more stake = higher probability.
// It's deterministic: given the same staked balances and the same seed,
// it always picks the same validator. The seed is the previous block's hash,
// which every node agrees on.
func SelectValidator(state *core.State, seed string) (string, error) {
	// Build a sorted list of eligible validators
	type candidate struct {
		Address string
		Stake   uint64
	}

	var candidates []candidate
	var totalStake uint64

	for addr, stake := range state.StakedBalances {
		if stake >= MinStake {
			candidates = append(candidates, candidate{addr, stake})
			totalStake += stake
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no eligible validators (need at least %d staked tokens)", MinStake)
	}

	// Sort by address for deterministic ordering — map iteration order is random in Go
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Address < candidates[j].Address
	})

	// Convert the seed (prev block hash) into a number
	seedHash := sha256.Sum256([]byte(seed))
	randVal := binary.BigEndian.Uint64(seedHash[:8]) % totalStake

	// Walk through validators: whoever's stake range contains randVal wins
	var cumulative uint64
	for _, c := range candidates {
		cumulative += c.Stake
		if randVal < cumulative {
			return c.Address, nil
		}
	}

	// Should never reach here, but return last candidate as fallback
	return candidates[len(candidates)-1].Address, nil
}

// CreateBlockReward returns a coinbase transaction paying the validator.
func CreateBlockReward(validator string) core.Transaction {
	return core.NewTransaction("", validator, BlockReward)
}
