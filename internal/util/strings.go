package util

import "strings"

func SanitizeFilename(value string) string {
	invalid := `\/:*?"<>|`
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(invalid, r) {
			return '_'
		}
		return r
	}, value)
}

func CleanBaseFilename(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "Stream ") {
		value = strings.TrimPrefix(value, "Stream ")
	}

	suffixes := []string{
		" | Free Listening on SoundCloud",
		" _ for free on SoundCloud",
		" on SoundCloud",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) {
			value = strings.TrimSuffix(value, suffix)
		}
	}
	return strings.TrimSpace(value)
}
