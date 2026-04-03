package core

import (
	"encoding/json"
	"fmt"
	"os"
)

// Genesis defines the initial state of the blockchain — who starts with what.
type Genesis struct {
	Balances map[string]uint64 `json:"balances"`
}

// DefaultGenesis returns a hardcoded starting state.
func DefaultGenesis() Genesis {
	return Genesis{
		Balances: map[string]uint64{
			"treasury": 1000000,
		},
	}
}

// LoadGenesis reads a genesis configuration from a JSON file.
func LoadGenesis(path string) (Genesis, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Genesis{}, fmt.Errorf("reading genesis file: %w", err)
	}

	var g Genesis
	if err := json.Unmarshal(data, &g); err != nil {
		return Genesis{}, fmt.Errorf("parsing genesis file: %w", err)
	}

	if len(g.Balances) == 0 {
		return Genesis{}, fmt.Errorf("genesis file must define at least one balance")
	}

	return g, nil
}
