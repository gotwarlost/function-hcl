package semtok

type TokenType string

// Token types predefined in the LSP spec
const (
	TokenTypeClass         TokenType = "class"
	TokenTypeComment       TokenType = "comment"
	TokenTypeEnum          TokenType = "enum"
	TokenTypeEnumMember    TokenType = "enumMember"
	TokenTypeEvent         TokenType = "event"
	TokenTypeFunction      TokenType = "function"
	TokenTypeInterface     TokenType = "interface"
	TokenTypeKeyword       TokenType = "keyword"
	TokenTypeMacro         TokenType = "macro"
	TokenTypeMethod        TokenType = "method"
	TokenTypeModifier      TokenType = "modifier"
	TokenTypeNamespace     TokenType = "namespace"
	TokenTypeNumber        TokenType = "number"
	TokenTypeOperator      TokenType = "operator"
	TokenTypeParameter     TokenType = "parameter"
	TokenTypeProperty      TokenType = "property"
	TokenTypeRegexp        TokenType = "regexp"
	TokenTypeString        TokenType = "string"
	TokenTypeStruct        TokenType = "struct"
	TokenTypeType          TokenType = "type"
	TokenTypeTypeParameter TokenType = "typeParameter"
	TokenTypeVariable      TokenType = "variable"
)

type TokenTypes []TokenType

func (tt TokenTypes) AsStrings() []string {
	types := make([]string, len(tt))

	for i, tokenType := range tt {
		types[i] = string(tokenType)
	}

	return types
}
