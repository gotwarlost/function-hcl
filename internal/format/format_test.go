package format

import (
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetFlags(0)
}

func TestFormatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "simple object literal",
			input: `
resource "foo" "bar" {
tags = {
name: "test"
env: "prod"
}
bar = "baz"
}
`,
			expected: `
resource "foo" "bar" {
  tags = {
    name = "test"
    env  = "prod"
  }
  bar = "baz"
}
`,
		},
		{
			name: "inline object literal",
			input: `
resource "foo" "bar" { 
tags = {name:"test",env:"prod"} 
}`,
			expected: `
resource "foo" "bar" {
  tags = { name = "test", env = "prod" }
}`,
		},
		{
			name: "simple ternary",
			input: `
resource "foo" "bar" {
  value = condition ? "yes" : "no"
}
`,
			expected: `
resource "foo" "bar" {
  value = condition ? "yes" : "no"
}
`,
		},
		{
			name: "simple for expr",
			input: `
resource "foo" "bar" {
  items = [for s in strings : s]
}
`,
			expected: `
resource "foo" "bar" {
  items = [for s in strings : s]
}
`,
		},
		{
			name: "ternary in object",
			input: `
resource "foo" "bar" {
  config = {
    name: "test",
    value: enabled ? "on" : "off"
  }
}
`,
			expected: `
resource "foo" "bar" {
  config = {
    name  = "test",
    value = enabled ? "on" : "off"
  }
}
`,
		},
		{
			name: "for expr in object",
			input: `
resource "foo" "bar" {
  config = {
    name: "test",
    items: [for x in list : x]
  }
}
`,
			expected: `
resource "foo" "bar" {
  config = {
    name  = "test",
    items = [for x in list : x]
  }
}
`,
		},
		{
			name: "nested object literals",
			input: `
resource "foo" "bar" {
  outer = {
    key1: "value1"
    inner: {
      key2: "value2"
      key37: "value3"
    }
  }
}
`,
			expected: `
resource "foo" "bar" {
  outer = {
    key1 = "value1"
    inner = {
      key2  = "value2"
      key37 = "value3"
    }
  }
}
`,
		},
		{
			name: "nested ternary",
			input: `
resource "foo" "bar" {
  config = {
    outer: cond1 ? "a" : "b",
    nested: cond2 ? (cond3 ? "x" : "y") : "z"
  }
}
`,
			expected: `
resource "foo" "bar" {
  config = {
    outer  = cond1 ? "a" : "b",
    nested = cond2 ? (cond3 ? "x" : "y") : "z"
  }
}
`,
		},
		{
			name: "for with object literal",
			input: `
resource "foo" "bar" {
  items = [for x in list : {name: x, value: "test"}]
}
`,
			expected: `
resource "foo" "bar" {
  items = [for x in list : { name = x, value = "test" }]
}
`,
		},
		{
			name: "complex mixed",
			input: `
resource "foo" "bar" {
  simple = {key: "value"}
  ternary = condition ? "yes" : "no"
  for_list = [for s in strings : s]
  complex = {
    name: "test",
    conditional: enabled ? "on" : "off",
    mapped: [for x in items : {id: x, active: x > 0 ? true : false}],
    nested: {
      deep: "value"
    }
  }
}
`,
			expected: `
resource "foo" "bar" {
  simple   = { key = "value" }
  ternary  = condition ? "yes" : "no"
  for_list = [for s in strings : s]
  complex = {
    name        = "test",
    conditional = enabled ? "on" : "off",
    mapped      = [for x in items : { id = x, active = x > 0 ? true : false }],
    nested = {
      deep = "value"
    }
  }
}
`,
		},
		{
			name: "multiple blocks",
			input: `
resource "foo" "bar" {
  some_attr = "value"
}

locals {
  obj = {key: "value"}
}
`,
			expected: `
resource "foo" "bar" {
  some_attr = "value"
}

locals {
  obj = { key = "value" }
}
`,
		},
		{
			name: "multiple objects in list",
			input: `
resource "foo" "bar" {
  items = [
    {name: "first", value: 1},
    {name: "second", value: 2}
  ]
}
`,
			expected: `
resource "foo" "bar" {
  items = [
    { name = "first", value = 1 },
    { name = "second", value = 2 }
  ]
}
`,
		},
		{
			name: "ternary with 2 literals",
			input: `
resource "foo" "bar" {
  config = enabled ? {name: "on", state: true} : {name: "off", state: foo? "bar": "baz"}
}
`,
			expected: `
resource "foo" "bar" {
  config = enabled ? { name = "on", state = true } : { name = "off", state = foo ? "bar" : "baz" }
}
`,
		},
		{
			name: "for obj expression",
			input: `
resource "foo" "bar" {
  map = {for k, v in items : k => v}
}
`,
			expected: `
resource "foo" "bar" {
  map = { for k, v in items : k => v }
}
`,
		},
		{
			name: "complex expressions",
			input: `
resource "foo" "bar" {
  x = 10
  y = "a < b ? a : b"
  map = {
	for k, v in items : k => [
       for item in v: v.foo if v.foo > 10
    ]
  }
}
`,
			expected: `
resource "foo" "bar" {
  x = 10
  y = "a < b ? a : b"
  map = {
    for k, v in items : k => [
      for item in v : v.foo if v.foo > 10
    ]
  }
}
`,
		},
		{
			name: "eol comments",
			input: `
locals {
  region = { name : "us-west-1" } // this is a comment
}
`,
			expected: `
locals {
  region = { name = "us-west-1" } // this is a comment
}
`,
		},
		{
			name: "eol comments 2",
			input: `
locals {
  region = "us-west-1" # this is a comment
}
`,
			expected: `
locals {
  region = "us-west-1" # this is a comment
}
`,
		},
		{
			name: "eol comments 3",
			input: `
locals {
  region = "us-west-1" /* this is a comment */
}
`,
			expected: `
locals {
  region = "us-west-1" /* this is a comment */
}
`,
		},
		{
			name: "eol comments 4",
			input: `
locals {
  foo = {name: "foo1" } /* this is a foo comment */
  bar = {name : x > 0 ? "bar1" : "bar2" } /* this is a bar comment */
}
`,
			expected: `
locals {
  foo = { name = "foo1" } /* this is a foo comment */
  bar = { name = x > 0 ? "bar1" : "bar2" } /* this is a bar comment */
}
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out := Source(test.input, Options{StandardizeObjectLiterals: true})
			// log.Println(out)
			e := strings.TrimSpace(test.expected)
			a := strings.TrimSpace(out)
			assert.Equal(t, e, a)
		})
	}
}
