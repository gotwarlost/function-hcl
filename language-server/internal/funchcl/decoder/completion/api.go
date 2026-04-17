// Package completion provides facilities for auto-complete and hover information.
package completion

import (
	"log"
	"os"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/funchcl/decoder"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	"github.com/hashicorp/hcl/v2"
)

// these variables control internal expression debugging but are only enabled in
// code to minimize perf impact, such that the compiler can optimize away all
// debug code paths when the variable is false.
var (
	debugCompletion = false
	dumpDebugSource = false
	debugLogger     = log.New(os.Stderr, "", 0)
)

// Completer provides completion and hover information.
type Completer struct {
	ctx           decoder.CompletionContext
	maxCandidates int
}

// New creates a Completer.
func New(ctx decoder.CompletionContext) *Completer {
	maxCandidates := 100
	if n := decoder.GetBehavior().MaxCompletionItems; n > 0 {
		maxCandidates = n
	}
	return &Completer{
		ctx:           ctx,
		maxCandidates: maxCandidates,
	}
}

// CompletionAt returns completion candidates for a given position in a file.
func (c *Completer) CompletionAt(filename string, pos hcl.Pos) (ret lang.Candidates, _ error) {
	list, err := c.startCompletion(filename, pos)
	if err != nil {
		return ret, err
	}
	complete := true
	if len(list) > c.maxCandidates {
		list = list[:c.maxCandidates]
		complete = false
	}
	return lang.Candidates{
		List:       list,
		IsComplete: complete,
	}, nil
}

// HoverAt returns hover data for a given position in a file.
func (c *Completer) HoverAt(filename string, pos hcl.Pos) (*lang.HoverData, error) {
	return c.doHover(filename, pos)
}
