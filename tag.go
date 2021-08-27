package tfe

type Tag struct {
	ID   string `jsonapi:"primary,tags"`
	Name string `jsonapi:"attr,name,omitempty"`
}
