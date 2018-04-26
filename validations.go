package tfe

import (
	"regexp"
)

var (
	// A regular expression used to validate common string ID patterns.
	reStringID = regexp.MustCompile(`^[a-zA-Z0-9\-\._]+$`)
)

// isStringID checks if the given string pointer is non-nil and contains a
// typical string identifier.
func validStringID(in *string) bool {
	return in != nil && reStringID.MatchString(*in)
}

// validString checks if the given input is present and non-empty.
func validString(in *string) bool {
	return in != nil && *in != ""
}
