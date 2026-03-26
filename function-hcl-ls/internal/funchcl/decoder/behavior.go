package decoder

// LangServerBehavior contains flags that influence how the language server behaves
// based on calling client identity. A singleton instance is initialized during the
// LSP initialize call and made accessible to feature implementations via Context.
type LangServerBehavior struct {
	MaxCompletionItems         int  // max completion items to return (0 means use default of 100)
	IndentMultiLineProposals   bool // when true, add leading spaces to multiple proposals based on current indent
	InnerBraceRangesForFolding bool // when true, ensure that folding range is the range not including braces
}

var defaultBehavior LangServerBehavior

// SetBehavior sets the singleton LangServerBehavior instance.
func SetBehavior(b LangServerBehavior) {
	defaultBehavior = b
}

// GetBehavior returns the singleton LangServerBehavior instance.
func GetBehavior() LangServerBehavior {
	return defaultBehavior
}
