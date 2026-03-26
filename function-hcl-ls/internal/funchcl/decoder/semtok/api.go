package semtok

import (
	"fmt"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang/semtok"
)

func TokensFor(ctx decoder.Context, filename string) ([]semtok.SemanticToken, error) {
	file, ok := ctx.HCLFileByName(filename)
	if !ok {
		return nil, fmt.Errorf("file %s not found", filename)
	}
	b, ok := ctx.FileBytesByName(filename)
	if !ok {
		return nil, fmt.Errorf("file %s not found", filename)
	}
	w := newWalker(file, b)
	return w.fileTokens(), nil
}
