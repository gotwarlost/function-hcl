package hclutils_test

import (
	"fmt"
	"testing"

	"github.com/crossplane-contrib/function-hcl/internal/evaluator/hclutils"
	"github.com/stretchr/testify/assert"
)

func TestIdentifier(t *testing.T) {
	tests := []struct {
		ident string
		want  bool
	}{
		{"foo", true},
		{"fooBar", true},
		{"_fooBar", true},
		{"foo-bar", true},
		{"-foo-bar", false},
		{"a b", false},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("index-%d", i), func(t *testing.T) {
			assert.Equal(t, test.want, hclutils.IsIdentifier(test.ident))
		})
	}
}
