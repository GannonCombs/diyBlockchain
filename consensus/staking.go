package consensus

import (
	"fmt"

	"github.com/gannoncombs/diyBlockchain/core"
)

// Slash penalizes a dishonest validator by removing a portion of their stake.
// In our simplified model, slashing removes half the validator's stake.
func Slash(state *core.State, validator string) error {
	stake := state.StakedBalances[validator]
	if stake == 0 {
		return fmt.Errorf("cannot slash %s: no staked balance", validator)
	}

	penalty := stake / 2
	if penalty == 0 {
		penalty = 1 // Always penalize at least 1 token
	}

	state.StakedBalances[validator] -= penalty
	// Slashed tokens are burned (removed from circulation), not redistributed
	return nil
}
