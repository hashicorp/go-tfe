package tfe

import (
	"reflect"
	"sort"
	"testing"
)

func TestOrganizations(t *testing.T) {
	client := testClient(t)

	org1, cleanupOrg1 := createOrganization(t, client)
	defer cleanupOrg1()
	org2, cleanupOrg2 := createOrganization(t, client)
	defer cleanupOrg2()

	// Get an initial list of the organizations for comparison.
	orgs, err := client.Organizations()
	if err != nil {
		t.Fatal(err)
	}

	expect := []*Organization{org1, org2}

	// Sort to ensure we are comparing in the right order
	sort.Stable(OrganizationNameSort(expect))
	sort.Stable(OrganizationNameSort(orgs))

	if !reflect.DeepEqual(orgs, expect) {
		t.Fatalf("\nExpect:\n%#v\n\nActual:\n%#v", expect, orgs)
	}
}

func TestOrganization(t *testing.T) {
	client := testClient(t)

	org, cleanup := createOrganization(t, client)
	defer cleanup()

	t.Run("when the org exists", func(t *testing.T) {
		result, err := client.Organization(org.Name)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(result, org) {
			t.Fatalf("\nExpect:\n%+v\n\nActual:\n%+v", org, result)
		}
	})

	t.Run("when the org does not exist", func(t *testing.T) {
		if _, err := client.Organization(randomString(t)); err == nil {
			t.Fatal("Expect error, got nil")
		}
	})
}

func TestModifyOrganization(t *testing.T) {
	client := testClient(t)

	t.Run("with valid parameters", func(t *testing.T) {
		org, cleanup := createOrganization(t, client)
		defer cleanup()

		input := &ModifyOrganizationInput{
			Name:           org.Name,
			Rename:         randomString(t),
			Email:          randomString(t) + "@tfe.local",
			SAMLOwnersRole: "ownerz",
		}

		output, err := client.ModifyOrganization(input)
		if err != nil {
			t.Fatal(err)
		}

		// Make sure we clean up the renamed org.
		defer client.DeleteOrganization(&DeleteOrganizationInput{
			Name: output.Organization.Name,
		})

		// Also get a fresh result from the API to ensure we get the
		// expected values back.
		refreshedOrg, err := client.Organization(input.Rename)
		if err != nil {
			t.Fatal(err)
		}

		for _, resultOrg := range []*Organization{
			output.Organization,
			refreshedOrg,
		} {
			if v := resultOrg.Name; v != input.Rename {
				t.Fatalf("Expect %q, got %q", input.Rename, v)
			}
			if v := resultOrg.Email; v != input.Email {
				t.Fatalf("Expect %q, got %q", input.Email, v)
			}
			if v := resultOrg.SAMLOwnersRole; v != input.SAMLOwnersRole {
				t.Fatalf("Expect %q, got %q", input.SAMLOwnersRole, v)
			}
		}
	})

	t.Run("with invalid parameters", func(t *testing.T) {
		org, cleanup := createOrganization(t, client)
		defer cleanup()

		input := &ModifyOrganizationInput{
			Name:  org.Name,
			Email: "nope",
		}

		if _, err := client.ModifyOrganization(input); err == nil {
			t.Fatal("Expect error, got nil")
		}
	})

	t.Run("when only updating a subset of fields", func(t *testing.T) {
		org, cleanup := createOrganization(t, client)
		defer cleanup()

		input := &ModifyOrganizationInput{
			Name:  org.Name,
			Email: randomString(t) + "@tfe.local",
		}

		output, err := client.ModifyOrganization(input)
		if err != nil {
			t.Fatal(err)
		}

		result := output.Organization
		if v := result.Name; v != org.Name {
			t.Fatalf("Expect %q, got %q", org.Name, v)
		}
		if v := result.Email; v != input.Email {
			t.Fatalf("Expect %q, got %q", input.Email, v)
		}
		if v := result.SAMLOwnersRole; v != org.SAMLOwnersRole {
			t.Fatalf("Expect %q, got %q", org.SAMLOwnersRole, v)
		}
	})
}
