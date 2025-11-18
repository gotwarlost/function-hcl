package format

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	outWriter   io.Writer = os.Stdout
	errorWriter io.Writer = os.Stderr
)

type FormatCmd struct {
	Check     bool
	Recursive bool
	Opts      Options
}

func (f *FormatCmd) Execute(args []string) error {
	files, err := f.collectFiles(args)
	if err != nil {
		return err
	}
	if len(files) == 1 && files[0] == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		ret := Source(string(b), f.Opts)
		_, _ = fmt.Fprintln(outWriter, ret)
		return nil
	}

	changes := 0
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		ret := Source(string(b), f.Opts)
		if ret != string(b) {
			changes++
			if f.Check {
				_, _ = fmt.Fprintln(errorWriter, file)
			} else {
				err = os.WriteFile(file, []byte(ret), 0o644)
				if err != nil {
					return err
				}
			}
		}
	}

	if changes > 0 && f.Check {
		return fmt.Errorf("%d unformatted files found", changes)
	}
	return nil
}

func (f *FormatCmd) collectFiles(args []string) ([]string, error) {
	if len(args) == 0 {
		args = []string{"."}
	}
	var files []string
	for _, arg := range args {
		if arg == "-" {
			if len(args) > 1 {
				return nil, fmt.Errorf("cannot mix stdin with other files")
			}
			return args, nil
		}
		argFiles, err := f.collectFileOrDir(arg)
		if err != nil {
			return nil, err
		}
		files = append(files, argFiles...)
	}
	return files, nil
}

func (f *FormatCmd) collectFileOrDir(input string) ([]string, error) {
	var hclFiles []string

	s, err := os.Stat(input)
	if err != nil {
		return nil, err
	}
	if !s.IsDir() {
		return []string{input}, nil
	}

	err = filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".hcl" {
			hclFiles = append(hclFiles, path)
		}
		if info.IsDir() && !f.Recursive && path != input {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}
	return hclFiles, nil
}
