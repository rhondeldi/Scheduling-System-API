package Utils

import "strings"

// ignores white spaces and case.
func HasSubString(main string, sub string) bool {
	processed_main := RemoveWhiteSpace(main)
	processed_main = strings.ToLower(processed_main)

	processed_sub := RemoveWhiteSpace(sub)
	processed_sub = strings.ToLower(processed_sub)

	return strings.Contains(processed_main, processed_sub)
}

func IsEqualStrCaseInsensitiveIgnoreWhiteSpace(a, b string) bool {
	processed_a := RemoveWhiteSpace(a)
	processed_a = strings.ToLower(processed_a)

	processed_b := RemoveWhiteSpace(b)
	processed_b = strings.ToLower(processed_b)

	return processed_a == processed_b
}
