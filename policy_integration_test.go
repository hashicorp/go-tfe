// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoliciesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest1, pTestCleanup1 := createPolicy(t, client, orgTest)
	defer pTestCleanup1()
	pTest2, pTestCleanup2 := createPolicy(t, client, orgTest)
	defer pTestCleanup2()

	opaOptions := PolicyCreateOptions{
		Kind:  OPA,
		Query: String("data.example.rule"),
		Enforce: []*EnforcementOptions{
			{
				Mode: EnforcementMode(EnforcementMandatory),
			},
		},
	}
	pTest3, pTestCleanup3 := createPolicyWithOptions(t, client, orgTest, opaOptions)
	defer pTestCleanup3()

	t.Run("without list options", func(t *testing.T) {
		pl, err := client.Policies.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.Contains(t, pl.Items, pTest1)
		assert.Contains(t, pl.Items, pTest2)
		assert.Contains(t, pl.Items, pTest3)

		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 3, pl.TotalCount)
	})

	t.Run("with pagination", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		pl, err := client.Policies.List(ctx, orgTest.Name, &PolicyListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)

		assert.Empty(t, pl.Items)
		assert.Equal(t, 999, pl.CurrentPage)
		assert.Equal(t, 3, pl.TotalCount)
	})

	t.Run("with search", func(t *testing.T) {
		// Search by one of the policy's names; we should get only that policy
		// and pagination data should reflect the search as well
		pl, err := client.Policies.List(ctx, orgTest.Name, &PolicyListOptions{
			Search: pTest1.Name,
		})
		require.NoError(t, err)

		assert.Contains(t, pl.Items, pTest1)
		assert.NotContains(t, pl.Items, pTest2)
		assert.NotContains(t, pl.Items, pTest3)
		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 1, pl.TotalCount)
	})

	t.Run("with filter by kind", func(t *testing.T) {
		pl, err := client.Policies.List(ctx, orgTest.Name, &PolicyListOptions{
			Kind: OPA,
		})
		require.NoError(t, err)

		assert.Contains(t, pl.Items, pTest3)
		assert.NotContains(t, pl.Items, pTest1)
		assert.NotContains(t, pl.Items, pTest2)
		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 1, pl.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ps, err := client.Policies.List(ctx, badIdentifier, nil)
		assert.Nil(t, ps)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestPoliciesCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with no kind", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name:        String(name),
			Description: String("A sample policy"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".sentinel"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Policies.Read(ctx, p.ID)
		require.NoError(t, err)

		for _, item := range []*Policy{
			p,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, Sentinel, item.Kind)
			assert.Equal(t, *options.Description, item.Description)
		}
	})

	t.Run("with valid options - Sentinel", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name:        String(name),
			Description: String("A sample policy"),
			Kind:        Sentinel,
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".sentinel"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Policies.Read(ctx, p.ID)
		require.NoError(t, err)

		for _, item := range []*Policy{
			p,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, options.Kind, item.Kind)
			assert.Nil(t, options.Query)
			assert.Equal(t, *options.Description, item.Description)
		}
	})

	t.Run("with valid options - OPA", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name:        String(name),
			Description: String("A sample policy"),
			Kind:        OPA,
			Query:       String("terraform.main"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".rego"),
					Mode: EnforcementMode(EnforcementMandatory),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Policies.Read(ctx, p.ID)
		require.NoError(t, err)

		for _, item := range []*Policy{
			p,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, options.Kind, item.Kind)
			assert.Equal(t, *options.Query, *item.Query)
			assert.Equal(t, *options.Description, item.Description)
		}
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, orgTest.Name, PolicyCreateOptions{
			Name: String(badIdentifier),
			Enforce: []*EnforcementOptions{
				{
					Path: String(badIdentifier + ".sentinel"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("when options has an invalid name - OPA", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, orgTest.Name, PolicyCreateOptions{
			Name:  String(badIdentifier),
			Kind:  OPA,
			Query: String("terraform.main"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(badIdentifier + ".rego"),
					Mode: EnforcementMode(EnforcementAdvisory),
				},
			},
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("when options is missing name", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, orgTest.Name, PolicyCreateOptions{
			Enforce: []*EnforcementOptions{
				{
					Path: String(randomString(t) + ".sentinel"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options is missing name - OPA", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, orgTest.Name, PolicyCreateOptions{
			Kind:  OPA,
			Query: String("terraform.main"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(randomString(t) + ".rego"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options is missing query - OPA", func(t *testing.T) {
		name := randomString(t)
		p, err := client.Policies.Create(ctx, orgTest.Name, PolicyCreateOptions{
			Name: String(name),
			Kind: OPA,
			Enforce: []*EnforcementOptions{
				{
					Path: String(randomString(t) + ".rego"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		})
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredQuery)
	})

	t.Run("when options is missing an enforcement-OPA", func(t *testing.T) {
		options := PolicyCreateOptions{
			Name:  String(randomString(t)),
			Kind:  OPA,
			Query: String("terraform.main"),
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredEnforce)
	})

	t.Run("when options is missing an enforcement-Sentinel", func(t *testing.T) {
		options := PolicyCreateOptions{
			Name: String(randomString(t)),
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredEnforce)
	})

	t.Run("when options is missing enforcement path-Sentinel", func(t *testing.T) {
		options := PolicyCreateOptions{
			Name: String(randomString(t)),
			Enforce: []*EnforcementOptions{
				{
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredEnforcementPath)
	})

	t.Run("when options is missing enforcement path-OPA", func(t *testing.T) {
		options := PolicyCreateOptions{
			Name:  String(randomString(t)),
			Kind:  OPA,
			Query: String("terraform.main"),
			Enforce: []*EnforcementOptions{
				{
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredEnforcementPath)
	})

	t.Run("when options is missing enforcement path", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name: String(name),
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".sentinel"),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredEnforcementMode)
	})

	t.Run("when options is missing enforcement mode-OPA", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name:  String(name),
			Kind:  OPA,
			Query: String("terraform.main"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".sentinel"),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredEnforcementMode)
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, badIdentifier, PolicyCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestPoliciesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createPolicy(t, client, orgTest)
	defer pTestCleanup()

	t.Run("when the policy exists without content", func(t *testing.T) {
		p, err := client.Policies.Read(ctx, pTest.ID)
		require.NoError(t, err)

		assert.Equal(t, pTest.ID, p.ID)
		assert.Equal(t, pTest.Name, p.Name)
		assert.Equal(t, pTest.PolicySetCount, p.PolicySetCount)
		assert.Empty(t, p.Enforce)
		assert.Equal(t, pTest.Organization.Name, p.Organization.Name)
	})

	err := client.Policies.Upload(ctx, pTest.ID, []byte(`main = rule { true }`))
	require.NoError(t, err)

	t.Run("when the policy exists with content", func(t *testing.T) {
		p, err := client.Policies.Read(ctx, pTest.ID)
		require.NoError(t, err)

		assert.Equal(t, pTest.ID, p.ID)
		assert.Equal(t, pTest.Name, p.Name)
		assert.Equal(t, pTest.Description, p.Description)
		assert.Equal(t, pTest.PolicySetCount, p.PolicySetCount)
		assert.NotEmpty(t, p.Enforce)
		assert.NotEmpty(t, p.Enforce[0].Path)
		assert.NotEmpty(t, p.Enforce[0].Mode)
		assert.Equal(t, pTest.Organization.Name, p.Organization.Name)
	})

	t.Run("when the policy does not exist", func(t *testing.T) {
		p, err := client.Policies.Read(ctx, "nonexisting")
		assert.Nil(t, p)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid policy ID", func(t *testing.T) {
		p, err := client.Policies.Read(ctx, badIdentifier)
		assert.Nil(t, p)
		assert.Equal(t, err, ErrInvalidPolicyID)
	})
}

func TestPoliciesUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("when updating with an existing path", func(t *testing.T) {
		pBefore, pBeforeCleanup := createUploadedPolicy(t, client, true, orgTest)
		defer pBeforeCleanup()

		require.Equal(t, 1, len(pBefore.Enforce))

		pAfter, err := client.Policies.Update(ctx, pBefore.ID, PolicyUpdateOptions{
			Enforce: []*EnforcementOptions{
				{
					Path: String(pBefore.Enforce[0].Path),
					Mode: EnforcementMode(EnforcementAdvisory),
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(pAfter.Enforce))

		assert.Equal(t, pBefore.ID, pAfter.ID)
		assert.Equal(t, pBefore.Name, pAfter.Name)
		assert.Equal(t, pBefore.Description, pAfter.Description)
		assert.Equal(t, pBefore.Enforce[0].Path, pAfter.Enforce[0].Path)
		assert.Equal(t, EnforcementAdvisory, pAfter.Enforce[0].Mode)
	})

	t.Run("when updating with a nonexisting path", func(t *testing.T) {
		// Weirdly enough pAfter is not equal to pBefore as updating
		// a nonexisting path causes the enforce mode to reset to the default
		// hard-mandatory
		t.Skip("see comment...")

		pBefore, pBeforeCleanup := createUploadedPolicy(t, client, true, orgTest)
		defer pBeforeCleanup()

		require.Equal(t, 1, len(pBefore.Enforce))
		pathBefore := pBefore.Enforce[0].Path
		modeBefore := pBefore.Enforce[0].Mode

		pAfter, err := client.Policies.Update(ctx, pBefore.ID, PolicyUpdateOptions{
			Enforce: []*EnforcementOptions{
				{
					Path: String("nonexisting"),
					Mode: EnforcementMode(EnforcementAdvisory),
				},
			},
		})
		require.NoError(t, err)

		require.Equal(t, 1, len(pAfter.Enforce))
		assert.Equal(t, pBefore, pAfter)
		assert.Equal(t, pathBefore, pAfter.Enforce[0].Path)
		assert.Equal(t, modeBefore, pAfter.Enforce[0].Mode)
	})

	t.Run("with a new description", func(t *testing.T) {
		pBefore, pBeforeCleanup := createUploadedPolicy(t, client, true, orgTest)
		defer pBeforeCleanup()

		pAfter, err := client.Policies.Update(ctx, pBefore.ID, PolicyUpdateOptions{
			Description: String("A brand new description"),
		})
		require.NoError(t, err)

		assert.Equal(t, pBefore.Name, pAfter.Name)
		assert.Equal(t, pBefore.Enforce, pAfter.Enforce)
		assert.NotEqual(t, pBefore.Description, pAfter.Description)
		assert.Equal(t, "A brand new description", pAfter.Description)
	})

	t.Run("with a new query", func(t *testing.T) {
		options := PolicyCreateOptions{
			Description: String("A sample OPA policy"),
			Kind:        OPA,
			Query:       String("data.example.rule"),
			Enforce: []*EnforcementOptions{
				{
					Mode: EnforcementMode(EnforcementMandatory),
				},
			},
		}
		pBefore, pBeforeCleanup := createUploadedPolicyWithOptions(t, client, true, orgTest, options)
		defer pBeforeCleanup()

		pAfter, err := client.Policies.Update(ctx, pBefore.ID, PolicyUpdateOptions{
			Query: String("terraform.policy1.deny"),
		})
		require.NoError(t, err)

		assert.Equal(t, pBefore.Name, pAfter.Name)
		assert.Equal(t, pBefore.Enforce, pAfter.Enforce)
		assert.NotEqual(t, *pBefore.Query, *pAfter.Query)
		assert.Equal(t, "terraform.policy1.deny", *pAfter.Query)
	})

	t.Run("update query when kind is not OPA", func(t *testing.T) {
		pBefore, pBeforeCleanup := createUploadedPolicy(t, client, true, orgTest)
		defer pBeforeCleanup()

		pAfter, err := client.Policies.Update(ctx, pBefore.ID, PolicyUpdateOptions{
			Query: String("terraform.policy1.deny"),
		})
		require.NoError(t, err)

		assert.Equal(t, pBefore.Name, pAfter.Name)
		assert.Equal(t, pBefore.Enforce, pAfter.Enforce)
		assert.Equal(t, Sentinel, pAfter.Kind)
		assert.Nil(t, pAfter.Query)
	})

	t.Run("without a valid policy ID", func(t *testing.T) {
		p, err := client.Policies.Update(ctx, badIdentifier, PolicyUpdateOptions{})
		assert.Nil(t, p)
		assert.Equal(t, err, ErrInvalidPolicyID)
	})
}

func TestPoliciesDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, _ := createPolicy(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Policies.Delete(ctx, pTest.ID)
		require.NoError(t, err)

		// Try loading the policy - it should fail.
		_, err = client.Policies.Read(ctx, pTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the policy does not exist", func(t *testing.T) {
		err := client.Policies.Delete(ctx, pTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the policy ID is invalid", func(t *testing.T) {
		err := client.Policies.Delete(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidPolicyID)
	})
}

func TestPoliciesUpload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	pTest, pTestCleanup := createPolicy(t, client, nil)
	defer pTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		err := client.Policies.Upload(ctx, pTest.ID, []byte(`main = rule { true }`))
		require.NoError(t, err)
	})

	t.Run("with empty content", func(t *testing.T) {
		err := client.Policies.Upload(ctx, pTest.ID, []byte{})
		require.NoError(t, err)
	})

	t.Run("without any content", func(t *testing.T) {
		err := client.Policies.Upload(ctx, pTest.ID, nil)
		require.NoError(t, err)
	})

	t.Run("without a valid policy ID", func(t *testing.T) {
		err := client.Policies.Upload(ctx, badIdentifier, []byte(`main = rule { true }`))
		assert.Equal(t, err, ErrInvalidPolicyID)
	})
}

func TestPoliciesDownload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	pTest, pTestCleanup := createPolicy(t, client, nil)
	defer pTestCleanup()

	testContent := []byte(`main = rule { true }`)

	t.Run("without existing content", func(t *testing.T) {
		content, err := client.Policies.Download(ctx, pTest.ID)
		assert.Equal(t, ErrResourceNotFound, err)
		assert.Nil(t, content)
	})

	t.Run("with valid options", func(t *testing.T) {
		err := client.Policies.Upload(ctx, pTest.ID, testContent)
		require.NoError(t, err)

		content, err := client.Policies.Download(ctx, pTest.ID)
		require.NoError(t, err)
		assert.Equal(t, testContent, content)
	})

	t.Run("without a valid policy ID", func(t *testing.T) {
		content, err := client.Policies.Download(ctx, badIdentifier)
		assert.Equal(t, err, ErrInvalidPolicyID)
		assert.Nil(t, content)
	})
}

func TestPolicy_Unmarshal(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "policies",
			"id":   "policy-ntv3HbhJqvFzamy7",
			"attributes": map[string]interface{}{
				"name":        "general",
				"description": "general policy",
				"enforce": []interface{}{
					map[string]interface{}{
						"path": "some/path",
						"mode": string(EnforcementAdvisory),
					},
				},
				"updated-at":       "2018-03-02T23:42:06.651Z",
				"policy-set-count": 1,
			},
		},
	}

	byteData, err := json.Marshal(data)
	require.NoError(t, err)

	responseBody := bytes.NewReader(byteData)
	policy := &Policy{}
	err = unmarshalResponse(responseBody, policy)
	require.NoError(t, err)

	iso8601TimeFormat := "2006-01-02T15:04:05Z"
	parsedTime, err := time.Parse(iso8601TimeFormat, "2018-03-02T23:42:06.651Z")
	require.NoError(t, err)
	assert.Equal(t, policy.ID, "policy-ntv3HbhJqvFzamy7")
	assert.Equal(t, policy.Name, "general")
	assert.Equal(t, policy.Description, "general policy")
	assert.Equal(t, policy.PolicySetCount, 1)
	assert.Equal(t, policy.Enforce[0].Path, "some/path")
	assert.Equal(t, policy.Enforce[0].Mode, EnforcementAdvisory)
	assert.Equal(t, policy.UpdatedAt, parsedTime)
}

func TestPolicyCreateOptions_Marshal(t *testing.T) {
	opts := PolicyCreateOptions{
		Name:        String("my-policy"),
		Description: String("details"),
		Enforce: []*EnforcementOptions{
			{
				Path: String("/foo"),
				Mode: EnforcementMode(EnforcementSoft),
			},
			{
				Path: String("/bar"),
				Mode: EnforcementMode(EnforcementSoft),
			},
		},
	}

	reqBody, err := serializeRequestBody(&opts)
	require.NoError(t, err)
	req, err := retryablehttp.NewRequest("POST", "url", reqBody)
	require.NoError(t, err)
	bodyBytes, err := req.BodyBytes()
	require.NoError(t, err)

	expectedBody := `{"data":{"type":"policies","attributes":{"description":"details","enforce":[{"path":"/foo","mode":"soft-mandatory"},{"path":"/bar","mode":"soft-mandatory"}],"name":"my-policy"}}}
`
	assert.Equal(t, expectedBody, string(bodyBytes))
}

func TestPolicyUpdateOptions_Marshal(t *testing.T) {
	opts := PolicyUpdateOptions{
		Description: String("details"),
		Enforce: []*EnforcementOptions{
			{
				Path: String("/foo"),
				Mode: EnforcementMode(EnforcementSoft),
			},
			{
				Path: String("/bar"),
				Mode: EnforcementMode(EnforcementSoft),
			},
		},
	}

	reqBody, err := serializeRequestBody(&opts)
	require.NoError(t, err)
	req, err := retryablehttp.NewRequest("POST", "url", reqBody)
	require.NoError(t, err)
	bodyBytes, err := req.BodyBytes()
	require.NoError(t, err)

	expectedBody := `{"data":{"type":"policies","attributes":{"description":"details","enforce":[{"path":"/foo","mode":"soft-mandatory"},{"path":"/bar","mode":"soft-mandatory"}]}}}
`
	assert.Equal(t, expectedBody, string(bodyBytes))
}
