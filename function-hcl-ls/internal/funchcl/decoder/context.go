// Package decoder supplies directory, file, and range level contexts for use by completion, hover, and other
// decoders.
package decoder

import (
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
)

// Context is the context under which expressions are evaluated. It provides schema and file information
// for module files.
type Context interface {
	schema.Lookup
	// Dir returns the directory path under which files are evaluated.
	Dir() string
	// Files is the list of filenames known at the current path.
	Files() []string
	// HCLFile returns the HCL file associated with the supplied expression.
	HCLFile(expr hcl.Expression) *hcl.File
	// FileBytes provides the bytes for the file associated with the supplied expression.
	FileBytes(e hcl.Expression) []byte
	// HCLFileByName returns the HCL file for the supplied name
	HCLFileByName(name string) (*hcl.File, bool)
	// FileBytesByName provides the bytes for the supplied file.
	FileBytesByName(name string) ([]byte, bool)
	// Behavior returns the language server behavior flags for the current session.
	Behavior() LangServerBehavior
}

// CompletionFuncContext is the context passed to a completion hook.
type CompletionFuncContext struct {
	PathContext Context // the path context.
	Dir         string  // the module directory.
	Filename    string  // the filename from where the hook was called.
	Pos         hcl.Pos // the position from where the hook was called.
}

// CompletionFunc is the function signature for completion hooks.
type CompletionFunc func(ctx CompletionFuncContext, matchPrefix string) ([]lang.HookCandidate, error)

// CompletionContext is the context used for completion and hover. In addition to the path context,
// it can provide a schema for all variables that are visible from a given position in a file.
type CompletionContext interface {
	Context
	// CompletionFunc returns a completion function for the supplied hook, if available or nil otherwise.
	CompletionFunc(hookName string) CompletionFunc
	// TargetSchema is the schema for references collected under a single attribute schema tree.
	TargetSchema() *schema.AttributeSchema
}

// ContextProvider provides path contexts.
type ContextProvider interface {
	Paths() ([]lang.Path, error)              // the paths for which we have open files
	PathContext(p lang.Path) (Context, error) // the path context at a specific directory.
}

// CompletionContextProvider provides path and completion contexts.
type CompletionContextProvider interface {
	ContextProvider
	PathCompletionContext(p lang.Path, file string, pos hcl.Pos) (CompletionContext, error)
}
