package main

import (
	"io"
	"log"
	"os"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/crossplane-contrib/function-hcl/function-hcl/internal/crds"
	"github.com/spf13/cobra"
)

func extractCRDsCommand() *cobra.Command {
	var dir string
	var byObject, progress, warnings bool
	localObjectsName := "local-objects"

	c := &cobra.Command{
		Use:   "extract-crds",
		Short: "extract CRDs from the supplied YAML files or glob patterns and write to stdout",
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
				for _, pattern := range args {
					if pattern == "-" {
						readers = append(readers, os.Stdin)
						continue
					}
					matches, err := doublestar.FilepathGlob(pattern)
					if err != nil {
						return err
					}
					for _, match := range matches {
						st, err := os.Stat(match)
						if err != nil {
							return err
						}
						if st.IsDir() {
							log.Printf("skip dir: %s", match)
							continue
						}
						f, err := os.Open(match)
						if err != nil {
							return err
						}
						readers = append(readers, f)
						closers = append(closers, f)
					}
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
