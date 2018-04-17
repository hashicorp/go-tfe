package tfe

// String returns a pointer to the given string.
func String(in string) *string {
	return &in
}

// Bool returns a pointer to the given bool
func Bool(in bool) *bool {
	return &in
}
