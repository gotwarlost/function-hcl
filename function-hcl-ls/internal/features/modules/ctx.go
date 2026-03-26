package modules

import (
	"fmt"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder"
	ourschema "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/schema"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/target"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type pathCtx struct {
	schema.Lookup
	dir   string
	files map[string]*hcl.File
}

func (c *pathCtx) Dir() string {
	return c.dir
}

func (c *pathCtx) Files() []string {
	var ret []string
	for file := range c.files {
		ret = append(ret, file)
	}
	return ret
}

func (c *pathCtx) HCLFile(expr hcl.Expression) *hcl.File {
	f, _ := c.HCLFileByName(expr.Range().Filename)
	return f
}

func (c *pathCtx) HCLFileByName(name string) (*hcl.File, bool) {
	f, ok := c.files[name]
	return f, ok
}

func (c *pathCtx) FileBytes(e hcl.Expression) []byte {
	b, _ := c.FileBytesByName(e.Range().Filename)
	return b
}

func (c *pathCtx) FileBytesByName(name string) ([]byte, bool) {
	f, ok := c.files[name]
	if !ok {
		return nil, false
	}
	return f.Bytes, true
}

func (c *pathCtx) Behavior() decoder.LangServerBehavior {
	return decoder.GetBehavior()
}

type ctx struct {
	pathCtx
	completionFunctions map[string]decoder.CompletionFunc
	targetSchema        *schema.AttributeSchema
}

func (c *ctx) TargetSchema() *schema.AttributeSchema {
	return c.targetSchema
}

func (c *ctx) CompletionFunc(hookName string) decoder.CompletionFunc {
	return c.completionFunctions[hookName]
}

func (m *Modules) Paths() ([]lang.Path, error) {
	recs := m.store.ListDirs()
	var ret []lang.Path
	for _, rec := range recs {
		ret = append(ret, lang.Path{
			Path:       rec,
			LanguageID: ourschema.LanguageHCL,
		})
	}
	return ret, nil
}

func (m *Modules) pathContext(p lang.Path) (decoder.Context, error) {
	rec := m.store.Get(p.Path)
	if rec == nil {
		return nil, fmt.Errorf("module not found at path: %s", p.Path)
	}
	return &pathCtx{
		dir:    p.Path,
		Lookup: ourschema.New(m.provider(p.Path)),
		files:  rec.Files,
	}, nil
}

// dynamicModuleLookup extends DynamicLookup to implement local variable and
// composite schema lookups.
type dynamicModuleLookup struct {
	ourschema.DynamicLookup
	targetSchema    *schema.AttributeSchema
	compositeSchema *schema.AttributeSchema
}

func (d *dynamicModuleLookup) LocalSchema(name string) *schema.AttributeSchema {
	cons, ok := d.targetSchema.Constraint.(schema.Object)
	if !ok {
		return nil
	}
	return cons.Attributes[name]
}

func (d *dynamicModuleLookup) CompositeSchema() *schema.AttributeSchema {
	return d.compositeSchema
}

var _ ourschema.LocalsAttributeLookup = &dynamicModuleLookup{}

func (m *Modules) pathCompletionContext(p lang.Path, filename string, pos hcl.Pos) (decoder.CompletionContext, error) {
	rec := m.store.Get(p.Path)
	if rec == nil {
		return nil, fmt.Errorf("module not found at path: %s", p.Path)
	}
	files := rec.Files
	file := files[filename]
	if file == nil {
		return nil, fmt.Errorf("module file %q not found", filename)
	}
	block := file.Body.(*hclsyntax.Body).InnermostBlockAtPos(pos)

	dyn := m.provider(p.Path)
	targets := rec.Targets
	if rec.XRD != nil && targets.CompositeSchema == nil {
		// we haven't used a composite schema previously; maybe it was still loading
		// see if we can redo this correctly.
		compositeSchema := dyn.Schema(rec.XRD.APIVersion, rec.XRD.Kind)
		if compositeSchema != nil {
			targets = target.BuildTargets(rec.Files, dyn, compositeSchema)
			rec.Targets = targets
			// note that even though the ReferenceMap depends on targets
			// it will not change just because we added a schema so no need
			// to recompute this.
			m.store.Put(rec)
		}
	}
	visibleTargets := targets.VisibleTreeAt(block, filename, pos)
	targetSchema := visibleTargets.AsSchema()
	return &ctx{
		pathCtx: pathCtx{
			dir: p.Path,
			Lookup: ourschema.New(&dynamicModuleLookup{
				DynamicLookup:   dyn,
				targetSchema:    targetSchema,
				compositeSchema: targets.CompositeSchema,
			}),
			files: files,
		},
		completionFunctions: map[string]decoder.CompletionFunc{
			"apiVersion": m.apiVersionCompletion,
			"kind":       m.kindCompletion,
		},
		targetSchema: targetSchema,
	}, nil
}

func (m *Modules) referenceMap(p lang.Path) (*target.ReferenceMap, error) {
	rec := m.store.Get(p.Path)
	if rec == nil {
		return nil, fmt.Errorf("module not found at path: %s", p.Path)
	}
	return rec.RefMap, nil
}
