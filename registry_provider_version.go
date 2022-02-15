package tfe

// RegistryProviderVersion represents a registry provider version
type RegistryProviderVersion struct {
	ID        string   `jsonapi:"primary,registry-provider-versions"`
	Version   string   `jsonapi:"attr,version"`
	KeyID     string   `jsonapi:"attr,key-id"`
	Protocols []string `jsonapi:"attr,protocols,omitempty"`

	// Relations
	RegistryProvider          *RegistryProvider          `jsonapi:"relation,registry-provider"`
	RegistryProviderPlatforms []RegistryProviderPlatform `jsonapi:"relation,registry-provider-platform"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}
