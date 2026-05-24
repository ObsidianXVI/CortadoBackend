package ai

import (
	"strings"
	"unicode/utf8"
)

func tailUTF8(input string, maxBytes int) string {
	if maxBytes <= 0 || len(input) <= maxBytes {
		return input
	}

	data := []byte(input)
	data = data[len(data)-maxBytes:]
	for len(data) > 0 && !utf8.Valid(data) {
		data = data[1:]
	}
	return string(data)
}

func headUTF8(input string, maxBytes int) string {
	if maxBytes <= 0 || len(input) <= maxBytes {
		return input
	}

	data := []byte(input[:maxBytes])
	for len(data) > 0 && !utf8.Valid(data) {
		data = data[:len(data)-1]
	}
	return string(data)
}

func lastNonEmptyLines(input string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}

	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) <= maxLines {
		return strings.Join(filtered, "\n")
	}
	return strings.Join(filtered[len(filtered)-maxLines:], "\n")
}
