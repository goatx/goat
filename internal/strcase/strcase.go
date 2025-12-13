package strcase

import (
	"strings"
	"unicode"
)

func ToCamelCase(name string) string {
	if name == "" {
		return name
	}

	firstChar := name[0]
	if firstChar >= 'A' && firstChar <= 'Z' {
		return string(firstChar+32) + name[1:]
	}

	return name
}

func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	var result []rune

	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				nextLower := false
				if i < len(runes)-1 {
					nextLower = unicode.IsLower(runes[i+1])
				}

				if unicode.IsLower(prev) || nextLower {
					result = append(result, '_')
				}
			}
			r = unicode.ToLower(r)
		}

		result = append(result, r)
	}

	return string(result)
}

func ToProtobufFieldName(s string) string {
	snake := ToSnakeCase(s)
	parts := strings.Split(snake, "_")
	var result strings.Builder
	for _, part := range parts {
		if part != "" {
			result.WriteString(strings.ToUpper(string(part[0])))
			result.WriteString(part[1:])
		}
	}
	return result.String()
}
