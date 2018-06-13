package tfe

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-uuid"
)

const badIdentifier = "! / nope"

func testClient(t *testing.T, fn ...func(*Config)) *Client {
	config := DefaultConfig()

	for _, f := range fn {
		f(config)
	}

	if v := os.Getenv("TFE_ADDRESS"); v != "" {
		config.Address = v
	}

	if config.Token == "" {
		config.Token = os.Getenv("TFE_TOKEN")
		if config.Token == "" {
			t.Fatal("TFE_TOKEN must be set")
		}
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func createConfigurationVersion(t *testing.T, client *Client, w *Workspace) (*ConfigurationVersion, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	cv, err := client.ConfigurationVersions.Create(
		w.ID,
		ConfigurationVersionCreateOptions{AutoQueueRuns: Bool(false)},
	)
	if err != nil {
		t.Fatal(err)
	}

	return cv, func() {
		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createUploadedConfigurationVersion(t *testing.T, client *Client, w *Workspace) (*ConfigurationVersion, func()) {
	cv, cvCleanup := createConfigurationVersion(t, client, w)

	fh, err := os.Open("test-fixtures/configuration-version.tar.gz")
	if err != nil {
		cvCleanup()
		t.Fatal(err)
	}
	defer fh.Close()

	if err := client.upload(cv.UploadURL, fh); err != nil {
		cvCleanup()
		t.Fatal(err)
	}

	for i := 0; ; i++ {
		cv, err = client.ConfigurationVersions.Retrieve(cv.ID)
		if err != nil {
			cvCleanup()
			t.Fatal(err)
		}

		if cv.Status == ConfigurationUploaded {
			break
		}

		if i > 10 {
			cvCleanup()
			t.Fatal("Timeout waiting for the configuration version to be uploaded")
		}

		time.Sleep(1 * time.Second)
	}

	return cv, cvCleanup
}

func createPolicy(t *testing.T, client *Client, org *Organization) (*Policy, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	name := randomString(t)
	options := PolicyCreateOptions{
		Name: String(name),
		Enforce: []*EnforcementOptions{
			&EnforcementOptions{
				Path: String(name + ".sentinel"),
				Mode: EnforcementMode(EnforcementSoft),
			},
		},
	}

	p, err := client.Policies.Create(org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	return p, func() {
		if err := client.Policies.Delete(p.ID); err != nil {
			t.Errorf("Error destroying policy! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Policy: %s\nError: %s", p.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createUploadedPolicy(t *testing.T, client *Client, pass bool, org *Organization) (*Policy, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	p, pCleanup := createPolicy(t, client, org)

	err := client.Policies.Upload(p.ID, []byte(fmt.Sprintf("main = rule { %t }", pass)))
	if err != nil {
		t.Fatal(err)
	}

	p, err = client.Policies.Retrieve(p.ID)
	if err != nil {
		t.Fatal(err)
	}

	return p, func() {
		pCleanup()

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createOAuthToken(t *testing.T, client *Client, org *Organization) (*OAuthToken, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	options := OAuthClientCreateOptions{
		APIURL:          String("https://api.github.com"),
		HTTPURL:         String("https://github.com"),
		Key:             String(randomString(t)),
		Secret:          String(randomString(t)),
		ServiceProvider: ServiceProvider(ServiceProviderGithub),
	}

	oc, err := client.OAuthClients.Create(org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	return oc.OAuthToken[0], func() {
		// There currently isn't a way to delete an OAuth client.
		//
		// if err := client.OAuthClients.Delete(oc.ID); err != nil {
		// 	t.Errorf("Error destroying OAuth client! WARNING: Dangling resources\n"+
		// 		"may exist! The full error is shown below.\n\n"+
		// 		"OAuthClient: %s\nError: %s", oc.ID, err)
		// }

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}
func createOrganization(t *testing.T, client *Client) (*Organization, func()) {
	org, err := client.Organizations.Create(OrganizationCreateOptions{
		Name:  String(randomString(t)),
		Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
	})
	if err != nil {
		t.Fatal(err)
	}

	return org, func() {
		if err := client.Organizations.Delete(org.Name); err != nil {
			t.Errorf("Error destroying organization! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Organization: %s\nError: %s", org.Name, err)
		}
	}
}

func createOrganizationToken(t *testing.T, client *Client, org *Organization) (*OrganizationToken, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	tk, err := client.OrganizationTokens.Generate(org.Name)
	if err != nil {
		t.Fatal(err)
	}

	return tk, func() {
		if err := client.OrganizationTokens.Delete(org.Name); err != nil {
			t.Errorf("Error destroying organization token! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"OrganizationToken: %s\nError: %s", tk.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	cv, cvCleanup := createUploadedConfigurationVersion(t, client, w)

	r, err := client.Runs.Create(RunCreateOptions{
		ConfigurationVersion: cv,
		Workspace:            w,
	})
	if err != nil {
		t.Fatal(err)
	}

	return r, func() {
		if wCleanup != nil {
			wCleanup()
		} else {
			cvCleanup()
		}
	}
}

func createPlannedRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	r, rCleanup := createRun(t, client, w)

	var err error
	for i := 0; ; i++ {
		r, err = client.Runs.Retrieve(r.ID)
		if err != nil {
			t.Fatal(err)
		}

		if r.Status == RunPlanned || r.Status == RunPolicyChecked || r.Status == RunPolicyOverride {
			break
		}

		if i > 30 {
			rCleanup()
			t.Fatal("Timeout waiting for run to be planned")
		}

		time.Sleep(1 * time.Second)
	}

	return r, rCleanup
}

func createSSHKey(t *testing.T, client *Client, org *Organization) (*SSHKey, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	key, err := client.SSHKeys.Create(org.Name, SSHKeyCreateOptions{
		Name:  String(randomString(t)),
		Value: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return key, func() {
		if err := client.SSHKeys.Delete(key.ID); err != nil {
			t.Errorf("Error destroying SSH key! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"SSHKey: %s\nError: %s", key.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createTeam(t *testing.T, client *Client, org *Organization) (*Team, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	tm, err := client.Teams.Create(org.Name, TeamCreateOptions{
		Name: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return tm, func() {
		if err := client.Teams.Delete(tm.ID); err != nil {
			t.Errorf("Error destroying team! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Team: %s\nError: %s", tm.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createTeamAccess(t *testing.T, client *Client, tm *Team, w *Workspace, org *Organization) (*TeamAccess, func()) {
	var orgCleanup, tmCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	if tm == nil {
		tm, tmCleanup = createTeam(t, client, org)
	}

	if w == nil {
		w, _ = createWorkspace(t, client, org)
	}

	ta, err := client.TeamAccesses.Add(TeamAccessAddOptions{
		Access:    Access(TeamAccessAdmin),
		Team:      tm,
		Workspace: w,
	})
	if err != nil {
		t.Fatal(err)
	}

	return ta, func() {
		if err := client.TeamAccesses.Remove(ta.ID); err != nil {
			t.Errorf("Error removing team access! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"TeamAccess: %s\nError: %s", ta.ID, err)
		}

		if tmCleanup != nil {
			tmCleanup()
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createTeamToken(t *testing.T, client *Client, tm *Team) (*TeamToken, func()) {
	var tmCleanup func()

	if tm == nil {
		tm, tmCleanup = createTeam(t, client, nil)
	}

	tk, err := client.TeamTokens.Generate(tm.ID)
	if err != nil {
		t.Fatal(err)
	}

	return tk, func() {
		if err := client.TeamTokens.Delete(tm.ID); err != nil {
			t.Errorf("Error destroying team token! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"TeamToken: %s\nError: %s", tm.ID, err)
		}

		if tmCleanup != nil {
			tmCleanup()
		}
	}
}

func createVariable(t *testing.T, client *Client, w *Workspace) (*Variable, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	v, err := client.Variables.Create(VariableCreateOptions{
		Key:       String(randomString(t)),
		Value:     String(randomString(t)),
		Category:  Category(CategoryTerraform),
		Workspace: w,
	})
	if err != nil {
		t.Fatal(err)
	}

	return v, func() {
		if err := client.Variables.Delete(v.ID); err != nil {
			t.Errorf("Error destroying variable! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Variable: %s\nError: %s", v.Key, err)
		}

		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createWorkspace(t *testing.T, client *Client, org *Organization) (*Workspace, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	w, err := client.Workspaces.Create(org.Name, WorkspaceCreateOptions{
		Name: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return w, func() {
		if err := client.Workspaces.Delete(org.Name, w.Name); err != nil {
			t.Errorf("Error destroying workspace! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Workspace: %s\nError: %s", w.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
