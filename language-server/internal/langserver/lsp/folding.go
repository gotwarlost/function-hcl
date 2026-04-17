package lsp

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/funchcl/decoder/folding"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
)

// FoldingRanges converts internal folding ranges to LSP folding ranges.
// Input ranges use 1-based line/column (HCL convention).
// Output ranges use 0-based line/column (LSP convention).
func FoldingRanges(ranges []folding.Range) []lsp.FoldingRange {
	result := make([]lsp.FoldingRange, 0, len(ranges))
	for _, r := range ranges {
		result = append(result, lsp.FoldingRange{
			StartLine:      uint32(r.StartLine - 1),
			StartCharacter: uint32(r.StartColumn - 1),
			EndLine:        uint32(r.EndLine - 1),
			EndCharacter:   uint32(r.EndColumn - 1),
			Kind:           r.Kind,
		})
	}
	return result
}
