package tfe

import (
	"regexp"
)

// TODO SvH: I would radar not check and test every input, but let the API
// return an error when it received something invalid. That way if things
// change we don't have to update this code and when people are trying to
// debug something using cURL they will see the same error response instead
// of our custom ones.
// If we do want to keep these, then let's create some default errors and
// return those instead of just a bool.

// A regular expression used to validate common string ID patterns.
var reStringID = regexp.MustCompile(`^[a-zA-Z0-9\-\._]+$`)

// validString checks if the given input is present and non-empty.
func validString(v *string) bool {
	return v != nil && *v != ""
}

// validStringID checks if the given string pointer is non-nil and
// contains a typical string identifier.
func validStringID(v *string) bool {
	return v != nil && reStringID.MatchString(*v)
}
