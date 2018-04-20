package tfe

// Permissions is used to model a set of permissions the current API user has
// on a given model within TFE.
type Permissions map[string]bool

// Can returns true if the user is allowed to perform the requested action.
func (p Permissions) Can(action string) bool {
	if p == nil {
		return false
	}
	return p["can-"+action]
}
