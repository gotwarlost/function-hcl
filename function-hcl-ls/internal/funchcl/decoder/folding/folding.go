// Package folding provides folding range support for HCL files.
// This implementation is dependent on the calling extension (vcode versus intellij)
// that expect slightly different ranges for folding behavior.
package folding

import (
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Range represents a foldable range in a document.
// Line and Column are 1-based (HCL convention).
type Range struct {
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	Kind        string // "comment", "imports", or "region"
}

// Collect returns all foldable ranges for the given HCL file.
// It folds blocks and object expressions with '{' delimiters.
// Tuples/lists with '[]' are not folded due to lsp4ij limitations.
func Collect(file *hcl.File, b decoder.LangServerBehavior) []Range {
	if file == nil {
		return nil
	}
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil
	}

	var ranges []Range

	// Walk the AST and collect foldable nodes
	_ = hclsyntax.VisitAll(body, func(node hclsyntax.Node) hcl.Diagnostics {
		switch n := node.(type) {
		case *hclsyntax.Block:
			if r, ok := blockFoldingRange(n, b.InnerBraceRangesForFolding); ok {
				ranges = append(ranges, r)
			}
		case *hclsyntax.ObjectConsExpr:
			if r, ok := objectFoldingRange(n, b.InnerBraceRangesForFolding); ok {
				ranges = append(ranges, r)
			}
		}
		return nil
	})

	return ranges
}

// blockFoldingRange returns a folding range for a block.
// The range starts right after the '{' and ends at the '}'.
// This satisfies lsp4ij's requirement that charAt(start-1) == '{' and charAt(end) == '}'.
func blockFoldingRange(block *hclsyntax.Block, innerBraces bool) (Range, bool) {
	if !isMultiline(block.Range()) {
		return Range{}, false
	}

	// Body range starts AT '{', we need position AFTER '{'
	bodyRange := block.Body.Range()

	// Block range ends after '}' - we need to position AT the '}'
	blockEnd := block.Range().End

	startCol := bodyRange.Start.Column
	endCol := bodyRange.End.Column
	if innerBraces {
		startCol++
		endCol--
	}

	return Range{
		StartLine:   bodyRange.Start.Line,
		StartColumn: startCol,
		EndLine:     blockEnd.Line,
		EndColumn:   endCol,
		Kind:        "region",
	}, true
}

// objectFoldingRange returns a folding range for an object expression.
// The range starts right after the '{' and ends at the '}'.
func objectFoldingRange(obj *hclsyntax.ObjectConsExpr, innerBraces bool) (Range, bool) {
	r := obj.Range()
	if !isMultiline(r) {
		return Range{}, false
	}

	startCol := r.Start.Column
	endCol := r.End.Column
	if innerBraces {
		startCol++
		endCol--
	}

	// Object range starts at '{', we need to start after it
	// Object range ends after '}', we need to position at the '}'
	return Range{
		StartLine:   r.Start.Line,
		StartColumn: startCol,
		EndLine:     r.End.Line,
		EndColumn:   endCol,
		Kind:        "region",
	}, true
}

func isMultiline(r hcl.Range) bool {
	return r.End.Line > r.Start.Line
}
