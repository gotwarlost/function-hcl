package functions

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type DynamicObject = map[string]cty.Value

type Arg struct {
	Name        string
	Description string
	HasDefault  bool
	Default     cty.Value
}

type UserFunction struct {
	Name         string
	Description  string
	Args         map[string]*Arg
	body         hcl.Expression
	blockContent *hcl.BodyContent
}

type Processor struct {
	Functions map[string]*UserFunction
	invoker   *invoker
}

func NewProcessor() *Processor {
	return &Processor{
		Functions: map[string]*UserFunction{},
		invoker:   newInvoker(nil),
	}
}

func (e *Processor) Process(content *hcl.BodyContent) hcl.Diagnostics {
	return e.processFunctions(content)
}

func (e *Processor) RootContext(values DynamicObject) *hcl.EvalContext {
	if values == nil {
		values = DynamicObject{}
	}
	return e.invoker.rootContext(values)
}
