package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/crossplane-contrib/function-hcl/function/internal/crds"
	"github.com/spf13/cobra"
)

func extractCRDsCommand() *cobra.Command {
	var dir string
	var byObject, progress, warnings bool
	localObjectsName := "local-objects"

	c := &cobra.Command{
		Use:   "extract-crds file.yaml [...file.yaml] -",
		Short: "extract CRDs from the supplied YAML files or glob patterns and write to stdout or files in an output directory",
		Long: `
extracts CRD and XRD definitions from the supplied YAML files or stdin and writes to stdout or a specified directory.
Additionally, image references in Provider and Configuration objects are also processed and additional CRDs are pulled
from those images.

YAML files can contain multiple documents. The ones that are used are CRDs and XRDs (as-is), crossplane Provider packages,
and crossplane configuration packages.

When no arguments are passed to the command, it acts a filter taking inputs over stdin and writing to stdout.

You can specify a - (dash) to explicitly request stdin processing. This argument can be mixed with other files.

When --output-dir (-o) is used, files are written to the output directory. The default behavior is to write one
multi-doc YAML file for each image that is processed, with a special image name to represent local objects in files.
This special name can be customized using the --local-objects-name option.

You can add the --by-object flag to create one file per object instead.

Warnings are produced by default when 2 versions of the same image are processed, or when duplicate CRDs or XRDs are
found. You can disable these using --warnings=false

Progress messages with timings are also produced per image. You can use --progress=false to turn them off.

`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			var readers []io.Reader
			var closers []io.Closer
			defer func() {
				for _, c := range closers {
					_ = c.Close()
				}
			}()

			if len(args) == 0 {
				readers = append(readers, os.Stdin)
			} else {
				for _, file := range args {
					if file == "-" {
						readers = append(readers, os.Stdin)
						continue
					}
					st, err := os.Stat(file)
					if err != nil {
						return err
					}
					if st.IsDir() {
						return fmt.Errorf("%s is a directory", file)
					}
					f, err := os.Open(file)
					if err != nil {
						return err
					}
					readers = append(readers, f)
					closers = append(closers, f)
				}
			}

			var w crds.Writer
			if dir != "" {
				if byObject {
					sw, err := crds.NewSplitFileWriter(dir)
					if err != nil {
						return err
					}
					w = sw
				} else {
					sw, err := crds.NewSplitImageWriter(dir)
					if err != nil {
						return err
					}
					w = sw
				}
			} else {
				w = crds.NewStreamWriter(os.Stdout)
			}

			l := log.New(os.Stderr, "", 0)
			if warnings {
				w = crds.NewMultiWriter(crds.NewWarningWriter(l), w)
			}
			if progress {
				w = crds.NewMultiWriter(crds.NewProgressWriter(l), w)
			}

			extractor := crds.NewExtractor(w, localObjectsName)
			err := extractor.ExtractCRDs(readers...)
			if err != nil {
				return err
			}
			return nil
		},
	}
	f := c.Flags()
	f.StringVarP(&dir, "output-dir", "o", "", "output directory to write files (defaults to stdout)")
	f.BoolVar(&byObject, "by-object", false, "write one file for every object, instead of one file per image (only used when output-dir is set)")
	f.BoolVar(&progress, "progress", true, "show progress information")
	f.BoolVar(&warnings, "warnings", true, "show duplicate image/ object warnings")
	f.StringVar(&localObjectsName, "local-objects-name", localObjectsName, "\"image name\" to use for objects found in local files (only used when writing one file per image)")
	return c
}
