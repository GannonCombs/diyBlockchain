package consensus

import (
	"testing"

	"github.com/gannoncombs/diyBlockchain/core"
)

// Helper: build a state with the given staked balances.
func stateWithStakes(stakes map[string]uint64) *core.State {
	return &core.State{
		Balances:       make(map[string]uint64),
		StakedBalances: stakes,
	}
}

func TestSelectValidatorDeterministic(t *testing.T) {
	state := stateWithStakes(map[string]uint64{
		"alice": 5000,
		"bob":   3000,
		"carol": 2000,
	})

	seed := "someblockhash123"

	// Same inputs should always produce the same result
	v1, _ := SelectValidator(state, seed)
	v2, _ := SelectValidator(state, seed)
	if v1 != v2 {
		t.Errorf("selection should be deterministic: got %s then %s", v1, v2)
	}
}

func TestSelectValidatorDifferentSeeds(t *testing.T) {
	state := stateWithStakes(map[string]uint64{
		"alice": 5000,
		"bob":   3000,
		"carol": 2000,
	})

	// Different seeds should (usually) produce different results.
	// Run many seeds and check we get more than one distinct validator.
	selected := make(map[string]bool)
	for i := 0; i < 100; i++ {
		seed := string(rune('a'+i)) + "seed"
		v, _ := SelectValidator(state, seed)
		selected[v] = true
	}

	if len(selected) < 2 {
		t.Errorf("expected multiple validators selected across 100 seeds, got %d", len(selected))
	}
}

func TestSelectValidatorWeightedFairness(t *testing.T) {
	// Alice has 90% of stake, bob has 10%
	state := stateWithStakes(map[string]uint64{
		"alice": 9000,
		"bob":   1000,
	})

	counts := map[string]int{"alice": 0, "bob": 0}
	trials := 1000
	for i := 0; i < trials; i++ {
		seed := string(rune(i%256)) + string(rune(i/256))
		v, _ := SelectValidator(state, seed)
		counts[v]++
	}

	// Alice should win roughly 90% of the time. Allow wide margin (70-100%).
	alicePct := float64(counts["alice"]) / float64(trials) * 100
	if alicePct < 70 || alicePct > 100 {
		t.Errorf("alice (90%% stake) selected %.0f%% of the time — expected ~90%%", alicePct)
	}
}

func TestSelectValidatorMinStake(t *testing.T) {
	state := stateWithStakes(map[string]uint64{
		"alice": 5000,
		"bob":   500, // Below MinStake (1000)
	})

	v, _ := SelectValidator(state, "anyseed")
	if v != "alice" {
		t.Errorf("bob is below MinStake, only alice should be selected, got %s", v)
	}
}

func TestSelectValidatorNoValidators(t *testing.T) {
	state := stateWithStakes(map[string]uint64{
		"alice": 100, // Below MinStake
	})

	_, err := SelectValidator(state, "anyseed")
	if err == nil {
		t.Error("should error when no eligible validators")
	}
}

func TestSlash(t *testing.T) {
	state := stateWithStakes(map[string]uint64{
		"alice": 10000,
	})

	err := Slash(state, "alice")
	if err != nil {
		t.Fatalf("slash failed: %s", err)
	}

	if state.StakedBalances["alice"] != 5000 {
		t.Errorf("alice should have 5000 after slashing, got %d", state.StakedBalances["alice"])
	}
}

func TestSlashNonValidator(t *testing.T) {
	state := stateWithStakes(map[string]uint64{})

	err := Slash(state, "nobody")
	if err == nil {
		t.Error("should error when slashing account with no stake")
	}
}
