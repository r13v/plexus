package codegraph

import (
	"github.com/odvcencio/gotreesitter/grammars"
)

// extractTS parses TypeScript or TSX source. The TS grammar is a superset of
// the JS grammar; we reuse extractJSLike and enable TS-only kinds via the
// isTS flag.
func extractTS(path string, src []byte, isTSX bool) (*FileExtraction, error) {
	lang := grammars.TypescriptLanguage()
	if isTSX {
		lang = grammars.TsxLanguage()
	}
	return extractJSLike(path, src, lang, true)
}
