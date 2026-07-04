// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	_ "embed"
	"log"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/cmd"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "function-hcl-ls",
		Short: "Language server for function-hcl",
	}
	cmd.Version = getBuildInfo().Version
	AddVersionCommand(root)
	cmd.AddServeCommand(root)
	cmd.AddDumpASTCommand(root)

	if err := root.Execute(); err != nil {
		log.Fatalln(err)
	}
}
