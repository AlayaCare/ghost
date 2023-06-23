package slugify

// from https://github.com/uretgec/slugify

import (
	"regexp"
	"strings"
)

// Make: generate from title to slug
func Make(str string, replacerMap ...string) string {
	str = strings.NewReplacer(makeReplacerMap(replacerMap)...).Replace(str)

	// remove non-alphanumeric chars
	strexp := regexp.MustCompile(`(?i)[^-a-zA-Z0-9\s]+`)
	str = strexp.ReplaceAllString(str, "")

	// convert spaces to dashes
	strexp = regexp.MustCompile(`(?i)\s`)
	str = strexp.ReplaceAllString(str, "-")

	// trim repeated dashes
	strexp = regexp.MustCompile(`(?i)[-]+`)
	str = strexp.ReplaceAllString(str, "-")

	return strings.ToLower(str)
}

func makeReplacerMap(replacerMap []string) []string {
	defaultReplacerMap := []string{
		`ç`, "c",
		`Ç`, "c",
		`ğ`, "g",
		`Ğ`, "g",
		`ş`, "s",
		`Ş`, "s",
		`ü`, "u",
		`Ü`, "u",
		`ı`, "i",
		`İ`, "i",
		`ö`, "o",
		`Ö`, "o",
	}

	if len(replacerMap) == 0 {
		return defaultReplacerMap
	}

	return append(defaultReplacerMap, replacerMap...)
}
