package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/spf13/cobra"
)

func typeString(node any, depth int) string {
	leader := strings.Repeat("  ", depth)
	t := strings.TrimPrefix(reflect.TypeOf(node).String(), "*")
	t = strings.TrimPrefix(t, "hclsyntax.")
	return leader + t
}

type counter struct {
	depth  int
	maxLen int
}

func (c *counter) setMax(t string) {
	if len(t) > c.maxLen {
		c.maxLen = len(t)
	}
}

func (c *counter) Enter(node hclsyntax.Node) hcl.Diagnostics {
	c.setMax(typeString(node, c.depth))
	c.depth++
	if se, ok := node.(*hclsyntax.ScopeTraversalExpr); ok {
		for _, t := range se.Traversal {
			c.setMax(typeString(t, c.depth+1))
		}
	}
	if se, ok := node.(*hclsyntax.RelativeTraversalExpr); ok {
		for _, t := range se.Traversal {
			c.setMax(typeString(t, c.depth+1))
		}
	}
	return nil
}

func (c *counter) Exit(_ hclsyntax.Node) hcl.Diagnostics {
	c.depth--
	return nil
}

type writer struct {
	source          []byte
	depth           int
	typeLength      int
	maxSourceLength int
}

func (w *writer) paddedString(s string) string {
	s += strings.Repeat(" ", w.typeLength-len(s))
	return s
}

func (w *writer) paddedTypeString(node any) string {
	return w.paddedString(typeString(node, w.depth))
}

func (w *writer) formatCode(r hcl.Range) string {
	b := r.SliceBytes(w.source)
	src := string(b)
	src = strings.ReplaceAll(src, "\n", " ")
	src = strings.ReplaceAll(src, "\r", "")
	src = strings.ReplaceAll(src, "\t", "  ")
	maxlen := w.maxSourceLength
	if len(src) > maxlen {
		half := maxlen/2 - 2
		start := src[:half]
		rest := src[half:]
		rest = rest[len(rest)-half:]
		src = start + " ... " + rest
	}
	return src
}

func (w *writer) WriteTraversal(traversal hcl.Traversal) {
	w.depth++
	for _, node := range traversal {
		fmt.Printf("%s : %s\n", w.paddedTypeString(node), w.formatCode(node.SourceRange()))
	}
	w.depth--
}

func (w *writer) Enter(node hclsyntax.Node) hcl.Diagnostics {
	lhs := w.paddedTypeString(node)
	src := w.formatCode(node.Range())
	fmt.Printf("%s : %s\n", lhs, src)
	w.depth++
	return nil
}

func (w *writer) Exit(node hclsyntax.Node) hcl.Diagnostics {
	if se, ok := node.(*hclsyntax.ScopeTraversalExpr); ok {
		w.WriteTraversal(se.Traversal)
	}
	if se, ok := node.(*hclsyntax.RelativeTraversalExpr); ok {
		w.WriteTraversal(se.Traversal)
	}
	w.depth--
	return nil
}

func printASTOutput(b []byte, expr bool) error {
	var e hclsyntax.Node
	var f *hcl.File
	var diags hcl.Diagnostics

	if expr {
		e, diags = hclsyntax.ParseExpression(b, "test.hcl", hcl.Pos{Line: 1, Column: 1})
	} else {
		f, diags = hclsyntax.ParseConfig(b, "test.hcl", hcl.Pos{Line: 1, Column: 1})
		if f != nil {
			e = f.Body.(*hclsyntax.Body)
		}
	}
	if diags.HasErrors() {
		log.Println("input has errors:", diags)
	}
	if e == nil {
		return nil
	}
	c := &counter{}
	_ = hclsyntax.Walk(e, c)
	_ = hclsyntax.Walk(e, &writer{
		source:          b,
		maxSourceLength: 60,
		typeLength:      c.maxLen,
	})
	return nil
}

// AddDumpASTCommand adds a sub-command to display the AST for HCL source.
func AddDumpASTCommand(root *cobra.Command) {
	var code string
	var expr bool
	c := &cobra.Command{
		Use:   "ast",
		Short: `display ast for an HCL expression read from stdin or option`,
	}
	root.AddCommand(c)
	f := c.Flags()
	f.StringVarP(&code, "code", "c", "", "code to evaluate, read from stdin if not supplied")
	f.BoolVarP(&expr, "expr", "e", false, "treat input as HCL expression rather than a source file")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		if code == "" {
			_, _ = fmt.Fprintf(os.Stderr, "reading stding for source...\n")
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			code = string(b)
		}
		return printASTOutput([]byte(code), expr)
	}
}
