// Package symbols provides document symbols.
package symbols

import (
	"fmt"
	"strings"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/funchcl/decoder"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	"github.com/hashicorp/hcl/v2"
)

type symbolImplSigil struct{}

// Symbol represents any attribute, or block (and its nested blocks or attributes)
type Symbol interface {
	Path() lang.Path
	Name() string
	NestedSymbols() []Symbol
	Range() hcl.Range
	isSymbolImpl() symbolImplSigil
}

// Collector collects symbols from documents.
type Collector struct {
	path lang.Path
}

// NewCollector returns a collector.
func NewCollector(path lang.Path) *Collector {
	return &Collector{path: path}
}

// FileSymbols find all symbols in a file as a hierarchical structure.
func (c *Collector) FileSymbols(ctx decoder.Context, filename string) ([]Symbol, error) {
	file, ok := ctx.HCLFileByName(filename)
	if !ok {
		return nil, fmt.Errorf("file %s not found", filename)
	}
	return c.symbolsForBody(file.Body), nil
}

// WorkspaceSymbols finds all symbols in all modules that are currently open.
// Note that this is buggy in that it doesn't load symbols from modules that are not
// being edited.
func WorkspaceSymbols(provider decoder.ContextProvider, query string) ([]Symbol, error) {
	var ret []Symbol
	paths, err := provider.Paths()
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		pc, err := provider.PathContext(p)
		if err != nil {
			return nil, err
		}
		for _, file := range pc.Files() {
			syms, err := NewCollector(p).FileSymbols(pc, file)
			if err != nil {
				return nil, err
			}
			for _, sym := range syms {
				if strings.Contains(sym.Name(), query) {
					ret = append(ret, sym)
				}
			}
		}
	}
	return ret, nil
}
