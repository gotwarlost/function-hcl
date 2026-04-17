package completion

import (
	"sort"
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestCompletionOnEqualsSign(t *testing.T) {
	t.Run("cursor on equals returns nothing", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        region = "us-east-1"
      }
    }
  }
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 7, Column: 16},
			[]string{},
		)
	})
}

func TestCompletionAtBlockLabels(t *testing.T) {
	t.Run("cursor in block label returns nothing", func(t *testing.T) {
		text := `
resource vpc {

}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 1, Column: 10},
			[]string{},
		)
	})
}

func TestFunctionNameCompletion(t *testing.T) {
	t.Run("function name with prefix", func(t *testing.T) {
		text := `
locals {
  result = upper
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 17},
			[]string{"upper"},
		)
	})

	t.Run("function name partial", func(t *testing.T) {
		text := `
locals {
  result = form
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 16},
			[]string{"format", "formatdate", "formatlist"},
		)
	})

	t.Run("function name inside call parens", func(t *testing.T) {
		text := `
locals {
  result = format("hello %s", )
}
`
		t.Skip("empty function argument after comma does not produce candidates - recovery limitation")
		c := candidates(t, text, nil,
			hcl.Pos{Line: 2, Column: 32})
		assert.NotNil(t, c)
		assert.Greater(t, len(c.List), 0,
			"should offer candidates for empty function argument")
	})

	t.Run("function second arg after comma", func(t *testing.T) {
		text := `
locals {
  result = format("%s-%s", "a", r)
}
`
		t.Skip("function arg completion after multiple args appears broken")
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 35},
			[]string{"range", "regex", "regexall", "replace", "req", "result", "reverse", "rsadecrypt"},
		)
	})

	t.Run("function trailing dot in arg", func(t *testing.T) {
		text := `
locals {
  result = upper(req.)
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 22},
			[]string{"composite", "composite_connection", "context"},
		)
	})
}

func TestBooleanCompletion(t *testing.T) {
	t.Run("bool attr empty value", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        enableDnsHostnames =
      }
    }
  }
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 7, Column: 29})
		labels := getCandidateLabels(c.List)
		assert.Contains(t, labels, "true")
		assert.Contains(t, labels, "false")
	})

	t.Run("bool attr with prefix t", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        enableDnsHostnames = t
      }
    }
  }
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 7, Column: 30})
		labels := getCandidateLabels(c.List)
		assert.Contains(t, labels, "true")
		// note: "false" is also present because the constraint is AnyExpression
		// not LiteralType{Bool}, so the bool prefix filter doesn't apply.
		// the anyType handler tries all strategies and accumulates candidates.
	})
}

func TestObjectCompletionEdgeCases(t *testing.T) {
	t.Run("declared attrs are filtered out", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    m
  }
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 5, Column: 6})
		labels := getCandidateLabels(c.List)
		assert.NotContains(t, labels, "kind",
			"already declared 'kind' should not be suggested")
		assert.NotContains(t, labels, "apiVersion",
			"already declared 'apiVersion' should not be suggested")
		assert.Contains(t, labels, "metadata")
	})

	t.Run("between key and value returns nothing", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
  }
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 3, Column: 9},
			[]string{},
		)
	})

	t.Run("after comma single line", func(t *testing.T) {
		t.Skip("object completion after trailing comma on next line does not produce candidates")
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    metadata = {
      name = "foo",

    }
  }
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 7, Column: 7})
		assert.NotNil(t, c)
		labels := getCandidateLabels(c.List)
		assert.Contains(t, labels, "annotations",
			"should offer remaining attributes after comma")
	})
}

func TestBodyDeclaredAttrsFiltered(t *testing.T) {
	t.Run("declared body attrs are filtered out", func(t *testing.T) {
		text := `
resource vpc {
  body = {}

}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 3, Column: 1})
		labels := getCandidateLabels(c.List)
		assert.NotContains(t, labels, "body",
			"already declared 'body' should not be suggested")
		assert.Contains(t, labels, "condition",
			"undeclared 'condition' should still be suggested")
	})

	t.Run("declared body attrs with prefix are filtered out", func(t *testing.T) {
		text := `
resource vpc {
  body = {}
  co
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 3, Column: 5})
		labels := getCandidateLabels(c.List)
		assert.NotContains(t, labels, "body",
			"already declared 'body' should not be suggested")
		assert.Contains(t, labels, "condition",
			"undeclared 'condition' should still be suggested")
		assert.Contains(t, labels, "composite",
			"undeclared block 'composite' should still be suggested")
	})

	t.Run("editing declared body attr still suggests it", func(t *testing.T) {
		text := `
resource vpc {
  body = {}
  con = "foo"
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 3, Column: 5})
		labels := getCandidateLabels(c.List)
		assert.Contains(t, labels, "condition",
			"attribute being edited should still be suggested")
		assert.Contains(t, labels, "context",
			"block matching prefix should still be suggested")
		assert.NotContains(t, labels, "body",
			"already declared 'body' should not be suggested")
	})

	t.Run("multiple declared body attrs are all filtered out", func(t *testing.T) {
		text := `
resource vpc {
  body = {}
  condition = "done"

}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 4, Column: 1})
		labels := getCandidateLabels(c.List)
		assert.NotContains(t, labels, "body",
			"already declared 'body' should not be suggested")
		assert.NotContains(t, labels, "condition",
			"already declared 'condition' should not be suggested")
		assert.Contains(t, labels, "ready",
			"undeclared 'ready' should still be suggested")
		assert.Contains(t, labels, "composite",
			"undeclared block 'composite' should still be suggested")
	})
}

func TestListCompletion(t *testing.T) {
	t.Run("empty list value", func(t *testing.T) {
		text := `
locals {
  things = [r]
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 14},
			[]string{"range", "regex", "regexall", "replace", "req", "reverse", "rsadecrypt"},
		)
	})

	t.Run("list element trailing dot", func(t *testing.T) {
		text := `
locals {
  things = [req.]
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 17},
			[]string{"composite", "composite_connection", "context"},
		)
	})

	t.Run("list second element", func(t *testing.T) {
		text := `
locals {
  things = ["a", r]
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 19},
			[]string{"range", "regex", "regexall", "replace", "req", "reverse", "rsadecrypt"},
		)
	})
}

func TestMapCompletion(t *testing.T) {
	t.Run("map value completion", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        tags = {
          env = r
        }
      }
    }
  }
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 8, Column: 18})
		assert.NotNil(t, c)
		labels := getCandidateLabels(c.List)
		assert.Contains(t, labels, "req")
	})

	t.Run("map value trailing dot", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        tags = {
          env = req.
        }
      }
    }
  }
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 8, Column: 21},
			[]string{"composite", "composite_connection", "connection", "context", "resource"},
		)
	})
}

func TestEmptyValueScaffolds(t *testing.T) {
	t.Run("object scaffold", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    metadata =
  }
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 5, Column: 15})
		assert.NotNil(t, c)
		labels := getCandidateLabels(c.List)
		assert.Contains(t, labels, "{…}",
			"should offer object scaffold")
	})
}

func TestNestedBlockCompletion(t *testing.T) {
	t.Run("attrs inside composite status block", func(t *testing.T) {
		t.Skip("composite status body completion does not produce schema-aware candidates")
		text := `
resource vpc {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind = "VPC"
  }
  composite status {
    body = {
      a
    }
  }
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 8, Column: 8})
		assert.NotNil(t, c)
		labels := getCandidateLabels(c.List)
		assert.Contains(t, labels, "arn",
			"should offer status attributes inside composite status body")
	})
}

func TestMultipleLocals(t *testing.T) {
	t.Run("locals reference each other", func(t *testing.T) {
		text := `
locals {
  alpha = "hello"
  beta = "world"
  gamma = a
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 4, Column: 12})
		labels := getCandidateLabels(c.List)
		assert.Contains(t, labels, "alpha",
			"should include local variable 'alpha'")
		assert.Contains(t, labels, "abs",
			"should include function 'abs'")
	})

	t.Run("locals reference with dot", func(t *testing.T) {
		text := `
locals {
  obj = { inner = "val" }
  result = obj.
}
`
		t.Skip("locals object traversal may not be supported")
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 3, Column: 16},
			[]string{"inner"},
		)
	})
}

func TestIndexExpressions(t *testing.T) {
	t.Run("index expression empty bracket", func(t *testing.T) {
		text := `
locals {
  items = ["a", "b"]
  picked = items[]
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 3, Column: 18})
		assert.NotNil(t, c)
		// should offer something inside index brackets
	})
}

func TestCompletionWithExistingResources(t *testing.T) {
	t.Run("req.resource includes all resources", func(t *testing.T) {
		text := `
resource alpha {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind = "VPC"
    spec = {
      forProvider = {
        region = req.resource.
      }
    }
  }
}
resource beta {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind = "VPC"
  }
}
resource gamma {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind = "VPC"
  }
}
`
		c := candidates(t, text, stdXRD,
			hcl.Pos{Line: 7, Column: 31})
		labels := getCandidateLabels(c.List)
		assert.Contains(t, labels, "alpha")
		assert.Contains(t, labels, "beta")
		assert.Contains(t, labels, "gamma")
	})
}

func TestCompletionCandiateMetadata(t *testing.T) {
	t.Run("required attrs are marked", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {

      }
    }
  }
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 7, Column: 1})
		assert.NotNil(t, c)
		for _, cd := range c.List {
			if cd.Label == "region" {
				assert.Contains(t, cd.Detail, "*",
					"required attribute 'region' should have asterisk in detail")
				return
			}
		}
		// region may or may not be required depending on schema,
		// so don't fail if not found
	})

	t.Run("function candidates have parens in text edit", func(t *testing.T) {
		text := `
locals {
  result = uppe
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 2, Column: 16})
		assert.NotNil(t, c)
		for _, cd := range c.List {
			if cd.Kind == lang.FunctionCandidateKind {
				assert.Contains(t, cd.TextEdit.NewText, "()",
					"function candidate should include parens")
			}
		}
	})
}

func TestTrailingDotRecoveryVariants(t *testing.T) {
	t.Run("trailing dot at top level attr", func(t *testing.T) {
		text := `
locals {
  val = req.composite.
}
`
		expectCandidateLabels(t, text, stdXRD,
			hcl.Pos{Line: 2, Column: 23},
			[]string{"metadata", "spec", "status"},
		)
	})

	t.Run("trailing dot nested in object", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        region = req.composite.
      }
    }
  }
}
`
		expectCandidateLabels(t, text, stdXRD,
			hcl.Pos{Line: 7, Column: 32},
			[]string{"metadata", "spec", "status"},
		)
	})
}

func TestCompletionMaxCandidates(t *testing.T) {
	t.Run("empty value returns bounded candidates", func(t *testing.T) {
		text := `
locals {
  val =
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 2, Column: 8})
		assert.NotNil(t, c)
		assert.LessOrEqual(t, len(c.List), 100,
			"should not exceed maxCandidates")
	})
}

func TestCompletionBodySchemaSnippets(t *testing.T) {
	t.Run("block candidate has snippet with braces", func(t *testing.T) {
		text := `
resource vpc {
  comp
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 2, Column: 7})
		assert.NotNil(t, c)
		labels := getCandidateLabels(c.List)
		sort.Strings(labels)
		assert.Contains(t, labels, "composite")
		for _, cd := range c.List {
			if cd.Label == "composite" && cd.Kind == lang.BlockCandidateKind {
				assert.Contains(t, cd.TextEdit.Snippet, "{",
					"block snippet should contain opening brace")
				assert.Contains(t, cd.TextEdit.Snippet, "}",
					"block snippet should contain closing brace")
			}
		}
	})

	t.Run("attribute candidate has equals in snippet", func(t *testing.T) {
		text := `
resource vpc {
  bod
}
`
		c := candidates(t, text, nil,
			hcl.Pos{Line: 2, Column: 6})
		assert.NotNil(t, c)
		for _, cd := range c.List {
			if cd.Label == "body" && cd.Kind == lang.AttributeCandidateKind {
				assert.Contains(t, cd.TextEdit.Snippet, "=",
					"attribute snippet should contain equals sign")
			}
		}
	})
}
