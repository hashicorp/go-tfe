package tfe

// NoCodeVariableOption represents a registry no-code module variable
// option.
type NoCodeVariableOption struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	Type string `jsonapi:"primary,variable-options"`

	// Required: The name of the variable
	VariableName string `jsonapi:"attr,variable-name"`

	// Required: The type of the variable
	VariableType string `jsonapi:"attr,variable-type"`

	Options []string `jsonapi:"attr,options"`
}
