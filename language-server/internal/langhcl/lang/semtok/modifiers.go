package semtok

type TokenModifier string

// Token modifiers predefined in the LSP spec
const (
	TokenModifierDeclaration    TokenModifier = "declaration"
	TokenModifierDefinition     TokenModifier = "definition"
	TokenModifierReadonly       TokenModifier = "readonly"
	TokenModifierStatic         TokenModifier = "static"
	TokenModifierDeprecated     TokenModifier = "deprecated"
	TokenModifierAbstract       TokenModifier = "abstract"
	TokenModifierAsync          TokenModifier = "async"
	TokenModifierModification   TokenModifier = "modification"
	TokenModifierDocumentation  TokenModifier = "documentation"
	TokenModifierDefaultLibrary TokenModifier = "defaultLibrary"
)

type TokenModifiers []TokenModifier

func (tm TokenModifiers) AsStrings() []string {
	modifiers := make([]string, len(tm))

	for i, tokenModifier := range tm {
		modifiers[i] = string(tokenModifier)
	}

	return modifiers
}
