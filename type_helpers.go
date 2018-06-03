package tfe

// AuthPolicy returns a pointer to the given authentication poliy.
func AuthPolicy(v AuthPolicyType) *AuthPolicyType {
	return &v
}

// Bool returns a pointer to the given bool
func Bool(v bool) *bool {
	return &v
}

// Delivery returns a pointer to the given delivery type.
func Delivery(v DeliveryType) *DeliveryType {
	return &v
}

// Int returns a pointer to the given bool
func Int(v int) *int {
	return &v
}

// ServiceProvider returns a pointer to the given service provider type.
func ServiceProvider(v ServiceProviderType) *ServiceProviderType {
	return &v
}

// String returns a pointer to the given string.
func String(v string) *string {
	return &v
}
