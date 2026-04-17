# function-hcl-ls - Modules and Flows

Document generate by Claude and is broadly correct as reviewed by a human.

## Modules Overview

### internal/langserver/ — LSP Server Framework

The entry point and request dispatch layer. Manages the JRPC2 server lifecycle (via `creachadair/jrpc2`), supporting both TCP and stdio transports. Concurrency is capped at CPU/2 on multi-CPU systems.

**Subpackages:**

- **handlers/**: LSP request handler implementations
  - `assign-handlers.go` — dispatch table mapping LSP methods to handlers
  - `server-lifecycle.go` — initialize, initialized, shutdown
  - `doc-lifecycle.go` — textDocument/didOpen, didChange, didClose
  - `doc-features.go` — completion, hover, definition, references, symbols, formatting
  - `diagnostics.go` — subscribes to diagnostic events and publishes them to the client
  - `crd_notification.go` — CRD configuration notifications
- **session/** — session lifecycle tracking (init state, context cancellation)
- **protocol/** — LSP protocol types
- **lsp/** — LSP utility functions for type conversions

### internal/eventbus/ — Type-Safe Event Pub/Sub

Decouples feature modules from document/server events using a generic topic-based publisher-subscriber pattern with channels (buffer size 10, thread-safe via `sync.Mutex`).

| Event               | Publisher                        | Subscriber      | Purpose                       |
|---------------------|----------------------------------|-----------------|-------------------------------|
| `OpenEvent`         | handlers (didOpen)               | modules feature | Track opened documents        |
| `EditEvent`         | handlers (didChange)             | modules feature | Trigger re-parse              |
| `ChangeWatchEvent`  | handlers (didChangeWatchedFiles) | modules feature | External file changes         |
| `DiagnosticsEvent`  | modules feature                  | handlers        | Push diagnostics to client    |
| `NoCRDSourcesEvent` | CRD feature                      | handlers        | Warn about missing CRD config |

### internal/document/ — Document Management

In-memory representation of open editor documents.

- **Document**: holds `Dir`, `Filename`, `Version`, `Text` (raw bytes), and `Lines` (pre-split for position calculations)
- **Handle / DirHandle**: efficient pointers to documents/directories
- **store/**: thread-safe map-based store (`sync.RWMutex`). Operations: Open, Update, Close, Get, List. Enforces strictly ascending version numbers.

### internal/filesystem/ — Overlayed Filesystem

Implements `io/fs.FS` with a two-layer system:

1. **Document Store layer** (preferred): returns unsaved editor content
2. **OS FS layer** (fallback): returns content from disk

Used by module parsing so that completion and diagnostics reflect unsaved changes.

### internal/features/modules/ — Module Parsing & Analysis

The core feature module. Subscribes to document events, queues per-directory parse jobs, and maintains parsed state.

**Processing pipeline:**

```
OpenEvent/EditEvent/ChangeWatchEvent
  → enqueue job to queue.Queue (keyed by directory)
  → fullParse() or incrementalParse()
  → parseModuleFile() + loadAndParseModule()
  → deriveData() → targets, refMap, diagnostics
  → store.Put(content)
  → publishDiagnostics() → DiagnosticsEvent
```

**Stored per directory (`store.Content`):**

- `Files` — `map[filename]*hcl.File` (parsed HCL ASTs)
- `Diags` — `map[filename]hcl.Diagnostics`
- `Targets` — symbol tree with scope info
- `RefMap` — bidirectional definition↔reference mappings
- `XRD` — optional `.xrd.yaml` metadata (apiVersion, kind)

**Key context types:**

- `PathContext` — full module context for a directory (implements `decoder.Context`)
- `CompletionContext` — extends PathContext with position-scoped visible variables and completion hooks

### internal/features/crds/ — CRD/XRD Schema Loading

Loads Kubernetes CRD and Crossplane XRD definitions and converts them to `AttributeSchema` structures.

- **store/**: background goroutine watching for CRD files, loads YAML, converts OpenAPIV3Schema recursively into langhcl schema constraints
- **cache.go**: CLI command to pre-download CRDs from OCI images
- Schema key format: `"apiVersion/kind"` (e.g., `"ec2.aws.upbound.io/v1beta1/Instance"`)

### internal/funchcl/decoder/ — Expression Parsing & Language Features

Provides the actual completion, hover, symbols, semantic tokens, and folding logic.

- **completion/**: `Completer` type with `CompletionAt()`, `HoverAt()`, `SignatureAtPos()`
  - `completion.go` — main algorithm: traverse body, find position, dispatch to expression-specific completers
  - `expr-completion.go` — generic expression completion
  - `expr-completion-ref.go` — reference traversals (`req.*`, `self.*`, `locals.*`)
  - `expr-completion-function.go` — function calls
  - `expr-completion-template.go` — string interpolation
  - `expr-completion-object.go` — object literals
  - `hover.go` / `hover-expr.go` — hover information extraction
  - `signature.go` — function signature help
- **symbols/**: AST walk to collect document symbols
- **semtok/**: AST traversal for semantic token generation
- **folding/**: code folding range collection

### internal/funchcl/target/ — Symbol Definition & Reference Tracking

Builds symbol trees and tracks definitions/references for go-to-definition and find-references.

- **Node**: tree node representing a symbol (name, schema, definition range, children)
- **Tree**: collection of root nodes with `VisibleTreeAt(block, file, pos)` for scoped visibility
- **Targets**: full symbol collection — globals, scoped locals, resource/collection aliases, composite schema
- **ReferenceMap**: bidirectional mapping with `FindDefinitionFromReference()` and `FindReferencesFromDefinition()`

Construction:

```
BuildTargets(files, dynamicSchemas, compositeSchema)
  → parse each file's targets (resource, locals, each, etc.)
  → merge into globals + scoped trees
  → build traversals (req.composite.*, self.*, etc.)

BuildReferenceMap(files, targets)
  → walk each expression
  → match traversals to definitions
  → bidirectional range mappings
```

### internal/funchcl/schema/ — Function-HCL DSL Schema

Provides schema lookup for HCL syntax and dynamic K8s type completion.

- **`Lookup`** — implements `schema.Lookup` with `BodySchema()`, `LabelSchema()`, `AttributeSchema()`, `Functions()`
- **`DynamicLookup`** — interface for runtime type schemas: `Schema(apiVersion, kind) → AttributeSchema`
- **`LocalsAttributeLookup`** — infer type from assignment (e.g., `foo = req.composite.name`)
- **`CompositeSchemaLookup`** — XRD/composite schema
- Built-in schemas for `resource`, `template`, `locals` blocks and standard functions

### internal/langhcl/schema/ — Generic Schema System

Language-agnostic schema description for any expression-based language.

**Constraint hierarchy** (all implement `Constraint` interface):

```
Constraint
├─ Scalar: String, Number, Bool, Literal
├─ Collection: List, Set, Map, Tuple
├─ Object {Attributes: map[string]*AttributeSchema}
├─ Any
└─ Reference {OfType: Constraint}
```

**AttributeSchema**: description, IsRequired/IsOptional, Constraint, CompletionHooks.

### internal/langhcl/lang/ — Generic Language Types

Language-agnostic types used across the completion/hover pipeline.

- **Candidate** — completion item (label, text edit, kind, description)
- **Candidates** — collection with `IsComplete` flag
- **HoverData** — hover content with markup
- **HookCandidate** — for custom completion hooks (apiVersion, kind)

---

## End-to-End Request Flows

### Session Initialization

```
LangServer.StartAndWait()
  → newService() → handlers.NewSession(ctx, version)
  → session.Service.Assigner() builds dispatch table

initialize request
  → configureSessionDependencies()
  → create docStore, filesystem, eventBus, CRDs, modules
  → CRDs.Start(ctx)          [background goroutine]
  → modules.Start(ctx)       [background goroutine]
  → startDiagnosticsPublisher(ctx) [background goroutine]

initialized notification
  → setupWatchedFiles() → register for workspace/didChangeWatchedFiles
```

Client behavior flags are set during initialize based on `clientInfo`:
- `MaxCompletionItems` (default 100, up to 1000 for IntelliJ)
- `InnerBraceRangesForFolding` (IntelliJ)
- `IndentMultiLineProposals` (IntelliJ)

### Completion (textDocument/completion)

```
Client → textDocument/completion
  │
  ├─ standardInit()
  │    docStore.Get(URI) → Document
  │    HCLPositionFromLspPosition(params.Position, doc)
  │
  ├─ modules.PathCompletionContext(path, filename, pos)
  │    Retrieve Content from store
  │    Parse HCL file
  │    Find InnermostBlockAtPos(pos)
  │    Get dynamic schemas via CRD provider
  │    Build targets with composite schema
  │    VisibleTreeAt(block, file, pos) → scoped variable tree
  │    Return CompletionContext with TargetSchema
  │
  ├─ completion.New(ctx).CompletionAt(filename, pos)
  │    bodyForFileAndPos() → navigate to enclosing body
  │    completeBodyAtPos(body, blockStack, pos)
  │      ├─ pos inside attribute value → attrValueCompletionAtPos()
  │      │    ├─ reference expr → expr-completion-ref.go (req.*, self.*, locals.*)
  │      │    ├─ function call → expr-completion-function.go
  │      │    ├─ template expr → expr-completion-template.go
  │      │    └─ object expr → expr-completion-object.go
  │      ├─ pos on attribute name → bodySchemaCandidates()
  │      └─ pos on block type/label → block-level completion
  │
  └─ ilsp.ToCompletionList(candidates, capabilities)
       → Client receives CompletionList
```

### Hover (textDocument/hover)

```
Client → textDocument/hover
  │
  ├─ standardInit() → doc, path, position
  ├─ modules.PathCompletionContext() → context with schemas
  │
  ├─ completion.New(ctx).HoverAt(filename, pos)
  │    doHover() → traverse to expression at pos
  │    ├─ attribute → schema.Description
  │    ├─ reference → traverse target tree, get description
  │    └─ function → signature + documentation
  │
  └─ ilsp.HoverData(hoverData, capabilities)
       → Client receives Hover with MarkupContent
```

### Go to Definition (textDocument/definition)

```
Client → textDocument/definition
  │
  ├─ standardInit() → doc, path, position
  │
  ├─ modules.ReferenceMap(path)
  │    Return pre-built ReferenceMap from store.Content
  │
  ├─ refMap.FindDefinitionFromReference(filename, pos)
  │    Iterate RefsToDef map
  │    Find range containing pos → definition range
  │
  └─ ilsp.ToLocationLinks(path, ranges)
       → Client receives LocationLink[]
```

### Find References (textDocument/references)

```
Client → textDocument/references
  │
  ├─ standardInit() → doc, path, position
  │
  ├─ modules.ReferenceMap(path)
  │
  ├─ refMap.FindReferencesFromDefinition(filename, pos)
  │    Find definition containing pos
  │    Return all reference ranges from DefToRefs
  │
  └─ ilsp.ToLocations(path, ranges)
       → Client receives Location[]
```

### Diagnostics (background, push-based)

```
Document event (open/edit/file change)
  → eventbus → modules feature
  │
  ├─ enqueue job per directory
  ├─ fullParse(dir)
  │    loadAndParseModule() → HCL parse errors
  │    deriveData() → semantic analysis
  │    analyze() → function-hcl specific checks
  │    Publish DiagnosticsEvent per file
  │
  └─ handlers.startDiagnosticsPublisher()
       Subscribe to DiagnosticsEvent
       Convert to LSP diagnostics
       → Client receives textDocument/publishDiagnostics
```

### Document Symbols (textDocument/documentSymbol)

```
Client → textDocument/documentSymbol
  │
  ├─ modules.PathContext(path) → full module context
  │
  ├─ symbols.NewCollector(path).FileSymbols(ctx, filename)
  │    Walk HCL body (hclsyntax)
  │    Collect blocks and attributes as symbols
  │    Build hierarchy
  │
  └─ ilsp.DocumentSymbols(syms, capabilities)
       → Client receives DocumentSymbol[]
```

### Semantic Tokens (textDocument/semanticTokens/full)

```
Client → textDocument/semanticTokens/full
  │
  ├─ modules.PathContext(path)
  │
  ├─ semtok.TokensFor(ctx, filename)
  │    Walk HCL AST (hclsyntax)
  │    Classify: keywords, identifiers, interpolations, operators
  │
  └─ ilsp.NewTokenEncoder(tokens, doc.Lines)
       Encode as delta positions
       → Client receives SemanticTokens{Data: [...]}
```

---

## Data Flow Diagram

```
┌──────────────────────────────────────────────────────────┐
│                    LSP Client (Editor)                    │
└──────────────────────────┬───────────────────────────────┘
                           │ JSON-RPC (stdio / TCP)
┌──────────────────────────▼───────────────────────────────┐
│                  langserver (JRPC2)                       │
│              handlers/assign-handlers.go                  │
└───┬──────────┬──────────┬──────────┬─────────────────────┘
    │          │          │          │
┌───▼───┐ ┌───▼────┐ ┌───▼───┐ ┌───▼────┐
│ doc   │ │ event  │ │ file  │ │  CRDs  │
│ store │ │  bus   │ │system │ │feature │
└───┬───┘ └───┬────┘ └───┬───┘ └───┬────┘
    │         │          │          │
    │    ┌────▼──────────▼──┐       │
    │    │  modules feature │       │
    │    │  (queue + store) │       │
    │    └──┬────────────┬──┘       │
    │       │            │          │
    │  ┌────▼─────┐ ┌───▼─────┐    │
    └──┤ decoder  │ │ target  │    │
       │completion│ │refs/tree│    │
       │hover     │ │         │    │
       │symbols   │ └─────────┘    │
       │semtok    │                │
       └────┬─────┘                │
            │                      │
       ┌────▼──────────────┐       │
       │  funchcl/schema   │◄──────┘
       │  (DynamicLookup)  │
       └────┬──────────────┘
            │
       ┌────▼──────────────┐
       │  langhcl/schema   │
       │  (Constraints)    │
       └───────────────────┘
```
