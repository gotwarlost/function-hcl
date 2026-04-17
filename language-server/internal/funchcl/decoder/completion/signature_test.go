package completion

import (
	"path/filepath"
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func signature(t *testing.T, text string, pos hcl.Pos) *lang.FunctionSignature {
	s := newTextScaffold(t, text, nil)
	ctx, updatedPos := s.completionContext(t, pos)
	completer := New(ctx)
	sig, err := completer.SignatureAtPos(filepath.Base(testFileName), updatedPos)
	require.NoError(t, err)
	return sig
}

func TestSignatureAtPos(t *testing.T) {
	t.Run("outside function call", func(t *testing.T) {
		text := `
locals {
  a = "hello"
}
`
		sig := signature(t, text, hcl.Pos{Line: 2, Column: 10})
		assert.Nil(t, sig)
	})

	t.Run("on function name", func(t *testing.T) {
		text := `
locals {
  a = upper("hi")
}
`
		// col 9 is on "p" in "upper", outside parens
		sig := signature(t, text, hcl.Pos{Line: 2, Column: 9})
		assert.Nil(t, sig)
	})

	t.Run("single param function", func(t *testing.T) {
		text := `
locals {
  a = upper("hi")
}
`
		// col 14 is on "h" in "hi"
		sig := signature(t, text, hcl.Pos{Line: 2, Column: 14})
		require.NotNil(t, sig)
		assert.Equal(t, "upper(str string) string", sig.Name)
		assert.NotEmpty(t, sig.Description.Value())
		require.Len(t, sig.Parameters, 1)
		assert.Equal(t, "str", sig.Parameters[0].Name)
		assert.Equal(t, uint32(0), sig.ActiveParameter)
	})

	t.Run("variadic function first arg", func(t *testing.T) {
		text := `
locals {
  a = format("%s", "b")
}
`
		// col 15 is on "%" in "%s" (first arg)
		sig := signature(t, text, hcl.Pos{Line: 2, Column: 15})
		require.NotNil(t, sig)
		assert.Equal(t, "format(format string, \u2026args dynamic) string", sig.Name)
		require.Len(t, sig.Parameters, 2)
		assert.Equal(t, "format", sig.Parameters[0].Name)
		assert.Equal(t, "args", sig.Parameters[1].Name)
		assert.Equal(t, uint32(0), sig.ActiveParameter)
	})

	t.Run("variadic function second arg", func(t *testing.T) {
		text := `
locals {
  a = format("%s", "b")
}
`
		// col 21 is on "b" (second arg)
		sig := signature(t, text, hcl.Pos{Line: 2, Column: 21})
		require.NotNil(t, sig)
		assert.Contains(t, sig.Name, "format")
		assert.Equal(t, uint32(1), sig.ActiveParameter)
	})

	t.Run("extra variadic args clamped to variadic index", func(t *testing.T) {
		text := `
locals {
  a = format("%s", "a", "b")
}
`
		// col 26 is on "b" (third arg, index 2)
		sig := signature(t, text, hcl.Pos{Line: 2, Column: 26})
		require.NotNil(t, sig)
		assert.Contains(t, sig.Name, "format")
		assert.Equal(t, uint32(1), sig.ActiveParameter,
			"should clamp to variadic parameter index")
	})

	t.Run("only variadic param first arg", func(t *testing.T) {
		text := `
locals {
  a = merge("a", "b")
}
`
		// col 14 is on "a" (first arg)
		sig := signature(t, text, hcl.Pos{Line: 2, Column: 14})
		require.NotNil(t, sig)
		assert.Equal(t, "merge(\u2026maps dynamic) dynamic", sig.Name)
		require.Len(t, sig.Parameters, 1)
		assert.Equal(t, "maps", sig.Parameters[0].Name)
		assert.Equal(t, uint32(0), sig.ActiveParameter)
	})

	t.Run("only variadic param second arg clamped", func(t *testing.T) {
		text := `
locals {
  a = merge("a", "b")
}
`
		// col 19 is on "b" (second arg, index 1)
		sig := signature(t, text, hcl.Pos{Line: 2, Column: 19})
		require.NotNil(t, sig)
		assert.Contains(t, sig.Name, "merge")
		assert.Equal(t, uint32(0), sig.ActiveParameter,
			"should clamp to variadic parameter index")
	})

	t.Run("too many args for non-variadic function", func(t *testing.T) {
		text := `
locals {
  a = upper("a", "b")
}
`
		// col 19 is on "b" (second arg for a single-param function)
		sig := signature(t, text, hcl.Pos{Line: 2, Column: 19})
		assert.Nil(t, sig,
			"should return nil when too many arguments for non-variadic function")
	})

	t.Run("nested function returns innermost signature", func(t *testing.T) {
		text := `
locals {
  a = upper(format("%s", "x"))
}
`
		// col 27 is on "x" inside format's second arg
		sig := signature(t, text, hcl.Pos{Line: 2, Column: 27})
		require.NotNil(t, sig)
		assert.Contains(t, sig.Name, "format",
			"should return signature for innermost function")
		assert.Equal(t, uint32(1), sig.ActiveParameter)
	})
}
