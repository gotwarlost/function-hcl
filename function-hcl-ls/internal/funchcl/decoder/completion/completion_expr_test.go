package completion

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestForExpressions(t *testing.T) {
	t.Run("complete for", func(t *testing.T) {
		text := `
locals {
  params = req.composite.spec.parameters
  cidrs = [for v in r ] 
}
`
		t.Skip("for expression completion doesn't yet work")
		expectCandidateLabels(t, text, stdXRD,
			hcl.Pos{Line: 3, Column: 22},
			[]string{"xxx"},
		)
	})
}

func TestConditionalExpressions(t *testing.T) {
	t.Run("conditional expression before question", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        region = b ? "x" : "y"
      }
    }
  }
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 7, Column: 19},
			[]string{"base64decode", "base64encode", "base64gzip", "base64sha256", "base64sha512"},
		)
	})

	t.Run("conditional expression after question", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        region = b ? "x" : "y"
      }
    }
  }
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 7, Column: 19},
			[]string{"base64decode", "base64encode", "base64gzip", "base64sha256", "base64sha512"},
		)
	})

	t.Run("conditional expression true branch", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        region = req.composite.spec.foo == "bar" ? r
      }
    }
  }
}
`
		t.Skip("fix this broken code, shows no completion at all!")
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 7, Column: 53},
			[]string{"xxx"},
		)
	})

	t.Run("conditional expression false branch", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        region = req.composite.spec.foo == "bar" ? "baz" : r
      }
    }
  }
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 7, Column: 61},
			[]string{"range", "regex", "regexall", "replace", "req", "reverse", "rsadecrypt"},
		)
	})

	t.Run("binary operator LHS", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        enableDnsHostnames = r == "enabled"
      }
    }
  }
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 7, Column: 31},
			[]string{"range", "regex", "regexall", "replace", "req", "reverse", "rsadecrypt"},
		)
	})

	t.Run("binary operator RHS", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        enableDnsHostnames = req.composite.spec.parameters.region == r
      }
    }
  }
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 7, Column: 71},
			[]string{"range", "regex", "regexall", "replace", "req", "reverse", "rsadecrypt"},
		)
	})

	t.Run("binary operator math", func(t *testing.T) {
		text := `
locals {
  base = 100
  scaled = base * r
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 3, Column: 20},
			[]string{"range", "regex", "regexall", "replace", "req", "reverse", "rsadecrypt"},
		)
	})

	t.Run("unary operator", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    kind = "VPC"
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    spec = {
      forProvider = {
        enableDnsHostnames = !r
      }
    }
  }
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 7, Column: 32},
			[]string{"range", "regex", "regexall", "replace", "req", "reverse", "rsadecrypt"},
		)
	})

	t.Run("unary operator dot", func(t *testing.T) {
		text := `
locals {
  enabled = !req.
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 18},
			[]string{"composite", "composite_connection", "context"},
		)
	})

	t.Run("parens", func(t *testing.T) {
		text := `
locals {
  result = (req.)
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 17},
			[]string{"composite", "composite_connection", "context"},
		)
	})

	t.Run("parens nested", func(t *testing.T) {
		text := `
locals {
  a = 10
  b = 20
  result = ((a + b) * r)
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 4, Column: 24},
			[]string{"range", "regex", "regexall", "replace", "req", "result", "reverse", "rsadecrypt"},
		)
	})

	t.Run("parens nested dot", func(t *testing.T) {
		text := `
locals {
  a = 10
  b = 20
  result = ((a + b) * req.)
}
`
		t.Skip("fix this broken code, shows no completion at all!")
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 4, Column: 27},
			[]string{"range", "regex", "regexall", "replace", "req", "result", "reverse", "rsadecrypt"},
		)
	})

	t.Run("func param", func(t *testing.T) {
		text := `
locals {
  result = format("%s", r)
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 26},
			[]string{"range", "regex", "regexall", "replace", "req", "result", "reverse", "rsadecrypt"},
		)
	})

	t.Run("func param dot", func(t *testing.T) {
		text := `
locals {
  result = format("%s", req.)
}
`
		expectCandidateLabels(t, text, nil,
			hcl.Pos{Line: 2, Column: 29},
			[]string{"composite", "composite_connection", "context"},
		)
	})
}
