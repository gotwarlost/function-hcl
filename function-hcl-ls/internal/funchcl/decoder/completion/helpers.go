package completion

import (
	"fmt"
	"strings"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (c *Completer) fileByName(name string) (*hcl.File, error) {
	f, ok := c.ctx.HCLFileByName(name)
	if !ok {
		return nil, &fileNotFoundError{filename: name}
	}
	return f, nil
}

func (c *Completer) bodyForFileAndPos(name string, f *hcl.File, pos hcl.Pos) (*hclsyntax.Body, error) {
	body := f.Body.(*hclsyntax.Body)

	//nolint:staticcheck
	if !(body.Range().ContainsPos(pos) ||
		posEqual(body.Range().Start, pos) ||
		posEqual(body.Range().End, pos)) {

		return nil, &posOutOfRangeError{
			filename: name,
			pos:      pos,
			rng:      body.Range(),
		}
	}
	return body, nil
}

func (c *Completer) bytesFromRange(rng hcl.Range) ([]byte, error) {
	b, err := c.bytesForFile(rng.Filename)
	if err != nil {
		return nil, err
	}
	return rng.SliceBytes(b), nil
}

func (c *Completer) bytesForFile(file string) ([]byte, error) {
	b, ok := c.ctx.FileBytesByName(file)
	if !ok {
		return nil, &fileNotFoundError{filename: file}
	}
	return b, nil
}

func (c *Completer) nameTokenRangeAtPos(filename string, pos hcl.Pos) (hcl.Range, error) {
	rng := hcl.Range{
		Filename: filename,
		Start:    pos,
		End:      pos,
	}

	f, err := c.fileByName(filename)
	if err != nil {
		return rng, err
	}
	tokens, diags := hclsyntax.LexConfig(f.Bytes, filename, hcl.InitialPos)
	if diags.HasErrors() {
		return rng, diags
	}

	return nameTokenRangeAtPos(tokens, pos)
}

func nameTokenRangeAtPos(tokens hclsyntax.Tokens, pos hcl.Pos) (hcl.Range, error) {
	// TODO: understand wtf is happening here
	for i, t := range tokens {
		if t.Range.ContainsPos(pos) {
			if t.Type == hclsyntax.TokenIdent {
				return t.Range, nil
			}
			if t.Type == hclsyntax.TokenNewline && i > 0 {
				// end of line
				previousToken := tokens[i-1]
				if previousToken.Type == hclsyntax.TokenIdent {
					return previousToken.Range, nil
				}
			}
			return hcl.Range{}, fmt.Errorf("token is %s, not Ident", t.Type.String())
		}

		// EOF token has zero length
		// so we just compare start/end position
		if t.Type == hclsyntax.TokenEOF && t.Range.Start == pos && t.Range.End == pos && i > 0 {
			previousToken := tokens[i-1]
			if previousToken.Type == hclsyntax.TokenIdent {
				return previousToken.Range, nil
			}
		}
	}
	return hcl.Range{}, fmt.Errorf("no token found at %s", posToStr(pos))
}

func isPosOutsideBody(block *hclsyntax.Block, pos hcl.Pos) bool {
	if block.OpenBraceRange.ContainsPos(pos) {
		return true
	}
	if block.CloseBraceRange.ContainsPos(pos) {
		return true
	}
	if hcl.RangeBetween(block.TypeRange, block.OpenBraceRange).ContainsPos(pos) {
		return true
	}
	return false
}

// detailForBlock returns a `Detail` info string to display in an editor in a hover event
func detailForBlock(_ *schema.BasicBlockSchema) string {
	detail := "Block"
	return strings.TrimSpace(detail)
}

// snippetForBlock takes a block and returns a formatted snippet for a user to complete inside an editor.
func snippetForBlock(blockType string, block *schema.BasicBlockSchema) string {
	labels := ""
	placeholder := 0
	for _, l := range block.Labels {
		placeholder++
		if l.CanComplete() {
			labels += fmt.Sprintf(` ${%d|%s|}`, placeholder, strings.Join(l.AllowedValues, ","))
		} else {
			labels += fmt.Sprintf(` ${%d:%s}`, placeholder, l.Name)
		}
	}
	return fmt.Sprintf("%s%s {\n  $0\n}", blockType, labels)
}
