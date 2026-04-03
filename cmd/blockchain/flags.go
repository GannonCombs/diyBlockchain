package main

import "flag"

// config holds parsed global flags.
type config struct {
	dataDir     string
	genesisPath string
	jsonOutput  bool
	// Per-command flags parsed on demand
	args []string
}

// parseGlobalFlags extracts --datadir, --genesis, and --json from args.
// Remaining unrecognized flags are kept in cfg.args for per-command parsing.
func parseGlobalFlags(args []string) config {
	fs := flag.NewFlagSet("global", flag.ContinueOnError)

	cfg := config{}
	fs.StringVar(&cfg.dataDir, "datadir", "./data", "")
	fs.StringVar(&cfg.genesisPath, "genesis", "./genesis.json", "")
	fs.BoolVar(&cfg.jsonOutput, "json", false, "")

	// Separate known global flags from command-specific flags.
	// We do a manual pass because flag.FlagSet doesn't support mixed args well.
	var remaining []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--datadir" && i+1 < len(args):
			cfg.dataDir = args[i+1]
			i++
		case args[i] == "--genesis" && i+1 < len(args):
			cfg.genesisPath = args[i+1]
			i++
		case args[i] == "--json":
			cfg.jsonOutput = true
		default:
			remaining = append(remaining, args[i])
		}
	}
	cfg.args = remaining
	return cfg
}
