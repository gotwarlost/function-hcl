package cmd

import (
	"fmt"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/features/crds"
	types "github.com/crossplane-contrib/function-hcl/function-hcl-ls/types/v1"
	"github.com/spf13/cobra"
)

// AddDownloadCRDsCommand adds a sub-command to download CRD definitions from package images,
// using the offline section of the CRD sources metadata file.
func AddDownloadCRDsCommand(root *cobra.Command) {
	c := &cobra.Command{
		Use:   "download-crds [<sources-file>]",
		Short: "download CRDs to local cache from a crd-sources file",
	}
	root.AddCommand(c)

	var deleteCache bool
	f := c.Flags()
	f.BoolVarP(&deleteCache, "delete-cache", "d", deleteCache, "delete cache dir before download")
	c.RunE = func(c *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("expected at most one argument")
		}
		file := types.StandardSourcesFile
		if len(args) == 1 {
			file = args[0]
		}
		c.SilenceUsage = true
		return crds.Download(file, deleteCache)
	}
}
