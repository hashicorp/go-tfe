// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validStackCreateOptions(t *testing.T) {
	t.Parallel()

	t.Run("with empty stack name", func(t *testing.T) {
		s := &StackCreateOptions{
			Name: "",
		}
		err := s.valid()
		assert.Error(t, err, ErrRequiredName.Error())
	})

	t.Run("with empty project option", func(t *testing.T) {
		s := &StackCreateOptions{
			Name: "test",
		}
		err := s.valid()
		assert.Error(t, err, ErrRequiredProject.Error())
	})

	t.Run("with empty project ID", func(t *testing.T) {
		s := &StackCreateOptions{
			Name: "test",
			Project: &Project{
				ID: "",
			},
		}
		err := s.valid()
		assert.Error(t, err, ErrRequiredProject.Error())
	})

	t.Run("with valid options", func(t *testing.T) {
		s := &StackCreateOptions{
			Name: "test",
			Project: &Project{
				ID: "prj-test",
			},
		}
		err := s.valid()
		assert.NoError(t, err)
	})
}
