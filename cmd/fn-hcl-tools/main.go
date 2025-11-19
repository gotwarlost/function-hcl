package main

import (
	"os"

	"github.com/spf13/cobra"
)

const exe = "fn-hcl-tools"

func main() {
	root := &cobra.Command{Use: exe}
	root.AddCommand(
		formatCommand(),
		analyzeCommand(),
		packageScriptCommand(),
		versionCommand(),
	)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
