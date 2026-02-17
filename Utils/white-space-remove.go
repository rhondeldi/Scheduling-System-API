package Utils

import (
	"strings"
	"unicode"
)

// remove white spaces using strings.Builder
func RemoveWhiteSpace(str string) string {

	var b strings.Builder

	b.Grow(len(str))

	for _, ch := range str {
		if !unicode.IsSpace(ch) {
			b.WriteRune(ch)
		}
	}

	return b.String()
}
