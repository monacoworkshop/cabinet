package str

import "strings"

// ToReplaceDashWithCamelCase ...
func ToReplaceDashWithCamelCase(s string) string {
	if len(s) == 1 {
		return strings.ToUpper(string(s[0]))
	}

	newString := strings.ToUpper(string(s[0]))
	for i := 1; i < len(s); i++ {
		if s[i] == '-' && i < len(s)-1 {
			i++
			newString += strings.ToUpper(string(s[i]))
			continue
		}

		newString += string(s[i])
	}
	return newString
}
