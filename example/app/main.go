package main

import (
	"github.com/reeflective/flags/gen/completions"
	"github.com/reeflective/flags/gen/flags"
)

func main() {
	//
	// Root ----------------------------------------------------------
	//
	rootData := &Command{}
	rootCmd := flags.Generate(rootData)
	rootCmd.SilenceUsage = true
	rootCmd.Short = "A local command demonstrating a few flags features"
	rootCmd.Long = "A longer help string used in detail help/usage output"

	// Completions (recursive)
	comps, _ := completions.Generate(rootCmd, rootData, nil)
	comps.Standalone()

	// Execute the command (application here)
	if err := rootCmd.Execute(); err != nil {
		return
	}
}
