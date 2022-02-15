package tfe

// RegistryProviderPlatform represents a registry provider platform
type RegistryProviderPlatform struct {
	ID       string `jsonapi:"primary,registry-provider-platforms"`
	Os       string `jsonapi:"attr,os"`
	Arch     string `jsonapi:"attr,arch"`
	Filename string `jsonapi:"attr,filename"`
	SHASUM   string `jsonapi:"attr,shasum"`

	// Relations
	RegistryProviderVersion *RegistryProviderVersion `jsonapi:"relation,registry-provider-version"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}
