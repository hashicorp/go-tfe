// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"testing"
)

// --------------------------------------------------------------------------
// Mock types
//
// These mirror the shape of Kiota-generated composed includedable types:
// each variant field is stored as an interface, so its zero value is the nil
// interface — the same condition FindSideloadedResource and
// FindAllSideloadedResources guard against with any(resource) == nil.
// --------------------------------------------------------------------------

// mockResource simulates a generated resource interface (e.g. models.Usersable).
type mockResource interface {
	GetId() *string
	GetEmail() *string
}

// mockResourceImpl is a concrete implementation used in tests.
type mockResourceImpl struct {
	id    *string
	email *string
}

func (m *mockResourceImpl) GetId() *string    { return m.id }
func (m *mockResourceImpl) GetEmail() *string { return m.email }

func newMockResource(id, email string) *mockResourceImpl {
	return &mockResourceImpl{id: &id, email: &email}
}

// mockIncluded simulates a Kiota composed includedable union type. Fields are
// declared as interfaces so their zero value is the nil interface, matching
// the behavior of generated code.
type mockIncluded struct {
	user mockResource // nil interface when this item is not a user
	team mockResource // nil interface when this item is not a team
}

func (m *mockIncluded) GetUser() mockResource { return m.user }
func (m *mockIncluded) GetTeam() mockResource { return m.team }

// extractUser is the extract callback for user resources.
func extractUser(item *mockIncluded) mockResource { return item.GetUser() }

// extractTeam is the extract callback for team resources.
func extractTeam(item *mockIncluded) mockResource { return item.GetTeam() }

// --------------------------------------------------------------------------
// Ptr
// --------------------------------------------------------------------------

func TestPtr(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		v := "hello"
		got := Ptr(v)
		if got == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *got != v {
			t.Errorf("expected %q, got %q", v, *got)
		}
	})

	t.Run("int", func(t *testing.T) {
		v := 42
		got := Ptr(v)
		if got == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *got != v {
			t.Errorf("expected %d, got %d", v, *got)
		}
	})

	t.Run("bool", func(t *testing.T) {
		got := Ptr(true)
		if got == nil || !*got {
			t.Errorf("expected pointer to true, got %v", got)
		}
	})

	t.Run("returns independent copy", func(t *testing.T) {
		original := "original"
		got := Ptr(original)
		original = "modified"
		if *got != "original" {
			t.Errorf("Ptr should capture a copy; got %q after modifying original", *got)
		}
	})

	t.Run("zero value", func(t *testing.T) {
		got := Ptr(0)
		if got == nil || *got != 0 {
			t.Errorf("expected pointer to 0, got %v", got)
		}
	})
}

// --------------------------------------------------------------------------
// Deref
// --------------------------------------------------------------------------

func TestDeref(t *testing.T) {
	t.Run("nil pointer returns default", func(t *testing.T) {
		var p *string
		got := Deref(p, "default")
		if got != "default" {
			t.Errorf("expected %q, got %q", "default", got)
		}
	})

	t.Run("non-nil pointer returns value", func(t *testing.T) {
		s := "hello"
		got := Deref(&s, "default")
		if got != "hello" {
			t.Errorf("expected %q, got %q", "hello", got)
		}
	})

	t.Run("int nil returns default", func(t *testing.T) {
		var p *int
		got := Deref(p, -1)
		if got != -1 {
			t.Errorf("expected -1, got %d", got)
		}
	})

	t.Run("int non-nil returns value", func(t *testing.T) {
		n := 99
		got := Deref(&n, 0)
		if got != 99 {
			t.Errorf("expected 99, got %d", got)
		}
	})

	t.Run("bool nil returns default false", func(t *testing.T) {
		var p *bool
		got := Deref(p, false)
		if got != false {
			t.Errorf("expected false, got %v", got)
		}
	})

	t.Run("bool non-nil true", func(t *testing.T) {
		b := true
		got := Deref(&b, false)
		if !got {
			t.Error("expected true")
		}
	})

	t.Run("zero value pointer is not nil", func(t *testing.T) {
		n := 0
		got := Deref(&n, 99)
		if got != 0 {
			t.Errorf("pointer to zero should return 0, got %d", got)
		}
	})
}

// --------------------------------------------------------------------------
// FindAllSideloadedResources
// --------------------------------------------------------------------------

func TestFindAllSideloadedResources(t *testing.T) {
	t.Run("empty included returns empty slice", func(t *testing.T) {
		got := FindAllSideloadedResources[*mockIncluded, mockResource](nil, extractUser)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %d items", len(got))
		}
	})

	t.Run("all items match the requested type", func(t *testing.T) {
		included := []*mockIncluded{
			{user: newMockResource("usr-1", "a@example.com")},
			{user: newMockResource("usr-2", "b@example.com")},
			{user: newMockResource("usr-3", "c@example.com")},
		}
		got := FindAllSideloadedResources(included, extractUser)
		if len(got) != 3 {
			t.Fatalf("expected 3 results, got %d", len(got))
		}
		expectIDs := []string{"usr-1", "usr-2", "usr-3"}
		for i, r := range got {
			if r.GetId() == nil || *r.GetId() != expectIDs[i] {
				t.Errorf("result[%d]: expected id %q, got %v", i, expectIDs[i], r.GetId())
			}
		}
	})

	t.Run("mixed types: only matching type returned", func(t *testing.T) {
		included := []*mockIncluded{
			{user: newMockResource("usr-1", "a@example.com")},
			{team: newMockResource("team-1", "")}, // user field is nil interface
			{user: newMockResource("usr-2", "b@example.com")},
			{team: newMockResource("team-2", "")},
		}
		got := FindAllSideloadedResources(included, extractUser)
		if len(got) != 2 {
			t.Fatalf("expected 2 user results, got %d", len(got))
		}
		if *got[0].GetId() != "usr-1" || *got[1].GetId() != "usr-2" {
			t.Errorf("unexpected IDs: %v, %v", *got[0].GetId(), *got[1].GetId())
		}
	})

	t.Run("no items match the requested type returns empty slice", func(t *testing.T) {
		included := []*mockIncluded{
			{team: newMockResource("team-1", "")},
			{team: newMockResource("team-2", "")},
		}
		got := FindAllSideloadedResources(included, extractUser)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %d items", len(got))
		}
	})

	t.Run("works for a different resource type (teams)", func(t *testing.T) {
		included := []*mockIncluded{
			{user: newMockResource("usr-1", "a@example.com")},
			{team: newMockResource("team-1", "")},
			{team: newMockResource("team-2", "")},
		}
		got := FindAllSideloadedResources(included, extractTeam)
		if len(got) != 2 {
			t.Fatalf("expected 2 team results, got %d", len(got))
		}
	})

	t.Run("items with nil id are still returned", func(t *testing.T) {
		// FindAllSideloadedResources does not filter by ID; that is
		// FindSideloadedResource's job. Resources with nil IDs are included
		// so the caller can inspect them.
		noID := &mockResourceImpl{id: nil, email: Ptr("noid@example.com")}
		included := []*mockIncluded{
			{user: noID},
		}
		got := FindAllSideloadedResources(included, extractUser)
		if len(got) != 1 {
			t.Fatalf("expected 1 result, got %d", len(got))
		}
		if got[0].GetId() != nil {
			t.Errorf("expected nil id, got %v", got[0].GetId())
		}
	})
}
