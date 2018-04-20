package tfe

import (
	"testing"
)

func TestPermissions(t *testing.T) {
	t.Run("when permissions are nil", func(t *testing.T) {
		var perm Permissions = nil
		if perm.Can("do-thing") {
			t.Fatal("expect false")
		}
	})

	t.Run("when the key does not exist", func(t *testing.T) {
		perm := Permissions{"can-nope": true}
		if perm.Can("do-thing") {
			t.Fatal("expect false")
		}
	})

	t.Run("when the key exists and is false", func(t *testing.T) {
		perm := Permissions{"can-do-thing": false}
		if perm.Can("do-thing") {
			t.Fatal("expect false")
		}
	})

	t.Run("when the key exists and is true", func(t *testing.T) {
		perm := Permissions{"can-do-thing": true}
		if !perm.Can("do-thing") {
			t.Fatal("expect true")
		}
	})
}
