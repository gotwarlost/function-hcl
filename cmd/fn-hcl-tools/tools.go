package main

import (
	"fmt"
	"log"
	"os"

	"github.com/crossplane-contrib/function-hcl/internal/evaluator"
	"github.com/crossplane-contrib/function-hcl/internal/format"
	"github.com/hashicorp/hcl/v2"
	"github.com/spf13/cobra"
	"golang.org/x/tools/txtar"
)

func doAnalyze(files []evaluator.File) error {
	e, err := evaluator.New(evaluator.Options{})
	if err != nil {
		return err
	}
	diags := e.Analyze(files...)
	for _, diag := range diags {
		sev := "ERROR:"
		if diag.Severity == hcl.DiagWarning {
			sev = "WARN :"
		}
		log.Println("\t", sev, diag.Error())
	}
	if diags.HasErrors() {
		return fmt.Errorf("analysis failed")
	}
	return nil
}

func analyzeCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "analyze file1.hcl file2.hcl ...",
		Short: "perform a static analysis of the supplied files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("no files to analyze")
			}
			cmd.SilenceUsage = true
			var files []evaluator.File
			for _, file := range args {
				contents, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				files = append(files, evaluator.File{
					Name:    file,
					Content: string(contents),
				})
			}
			return doAnalyze(files)
		},
	}
	return c
}

func packageScriptCommand() *cobra.Command {
	var skipAnalysis bool
	c := &cobra.Command{
		Use:   "package file1.hcl file2.hcl ...",
		Short: "generate a txtar script for the supplied files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("no files to package")
			}
			cmd.SilenceUsage = true
			var archive txtar.Archive
			var files []evaluator.File
			for _, file := range args {
				contents, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				archive.Files = append(archive.Files, txtar.File{
					Name: file,
					Data: contents,
				})
				files = append(files, evaluator.File{
					Name:    file,
					Content: string(contents),
				})
			}
			if !skipAnalysis {
				if err := doAnalyze(files); err != nil {
					return err
				}
			}
			b := txtar.Format(&archive)
			_, _ = os.Stdout.Write(b)
			return nil
		},
	}
	f := c.Flags()
	f.BoolVar(&skipAnalysis, "skip-analysis", false, "skip analysis of files before packaging")
	return c
}

func formatCommand() *cobra.Command {
	fc := format.FormatCmd{
		Check:     false,
		Recursive: true,
		Opts: format.Options{
			StandardizeObjectLiterals: true,
		},
	}
	c := &cobra.Command{
		Use:   "fmt file1.hcl file2.hcl dir/ ...",
		Short: "format HCL files",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return fc.Execute(args)
		},
	}
	f := c.Flags()
	f.BoolVar(&fc.Opts.StandardizeObjectLiterals, "normalize-literals", fc.Opts.StandardizeObjectLiterals, "normalize object literals to always use key = value syntax")
	f.BoolVarP(&fc.Check, "check", "c", fc.Check, "check if files are formatted, log names of unformatted files and exit appropriately")
	f.BoolVarP(&fc.Recursive, "recursive", "r", fc.Recursive, "recursively process directories")
	return c
}
