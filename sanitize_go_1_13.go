// +build go1.13

package thumbnailer

import (
	"strings"
	"unicode/utf8"
)

// Convert string to valid UTF-8.
// Strings passed from C are not guaranteed to be valid.
func sanitize(s *string) {
	if !utf8.ValidString(*s) {
		*s = strings.ToValidUTF8(*s, string(utf8.RuneError))
	}
}
