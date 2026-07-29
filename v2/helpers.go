// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

// Ptr returns a pointer to v. It eliminates the need for a temporary variable
// when a pointer to a non-addressable value is required, such as an enum
// literal or a function return value:
//
//	cfg := &abstractions.RequestConfiguration[T]{
//	    QueryParameters: &T{Include: tfe.Ptr(USER_GETINCLUDEQUERYPARAMETERTYPE)},
//	}
func Ptr[T any](v T) *T {
	return &v
}

// Deref safely dereferences ptr, returning the pointed-to value when non-nil
// or defaultVal when ptr is nil. It reduces nil-guard boilerplate for the
// pointer-typed fields that all generated models use:
//
//	email := tfe.Deref(user.GetAttributes().GetEmail(), "")
func Deref[T any](ptr *T, defaultVal T) T {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

// FindAllSideloadedResources retrieves every resource of a given type from the
// included sideloaded data. The extract callback identifies the type to collect
// and should return the appropriate GetXXX() call on the composed includedable
// item. Items for which extract returns nil are skipped (they carry a different
// resource type).
//
// To use this helper, your Get query must have used the Include query parameter
// to sideload the relationship. See FindSideloadedResource to look up a single
// resource by relationship ID instead.
//
// Example usage — collect all sideloaded users from an org-memberships response:
//
//	users := tfe.FindAllSideloadedResources(response.GetIncluded(),
//	    func(item organizations.ItemOrganizationMembershipsGetResponse_OrganizationMembershipsGetResponse_includedable) models.Usersable {
//	        return item.GetUsers()
//	    })
//	for _, u := range users {
//	    fmt.Println(tfe.Deref(u.GetAttributes().GetEmail(), "<no email>"))
//	}
func FindAllSideloadedResources[TIncluded any, TResource interface{ GetId() *string }](
	included []TIncluded,
	extract func(TIncluded) TResource,
) []TResource {
	var results []TResource
	for _, item := range included {
		resource := extract(item)
		// extract returns the nil interface value when the composed included item
		// does not carry the requested resource type. Because the generated code
		// stores each variant as an interface field (not a concrete pointer),
		// the nil-interface comparison here is correct: a zero-value interface
		// wrapped in any() is nil.
		if any(resource) == nil {
			continue
		}
		results = append(results, resource)
	}
	return results
}
