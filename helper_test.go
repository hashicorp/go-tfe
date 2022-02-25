package tfe

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-uuid"
)

const badIdentifier = "! / nope" //nolint

// Memoize test account details
var _testAccountDetails *TestAccountDetails

func testClient(t *testing.T) *Client {
	client, err := NewClient(nil)
	client.RetryServerErrors(true) // because occasionally we get a 500 internal when deleting an organization's workspace
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func fetchTestAccountDetails(t *testing.T, client *Client) *TestAccountDetails {
	if _testAccountDetails == nil {
		_testAccountDetails = FetchTestAccountDetails(t, client)
	}
	return _testAccountDetails
}

func createAgentPool(t *testing.T, client *Client, org *Organization) (*AgentPool, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	pool, err := client.AgentPools.Create(ctx, org.Name, AgentPoolCreateOptions{
		Name: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return pool, func() {
		if err := client.AgentPools.Delete(ctx, pool.ID); err != nil {
			t.Errorf("Error destroying agent pool! WARNING: Dangling resources "+
				"may exist! The full error is shown below.\n\n"+
				"Agent pool ID: %s\nError: %s", pool.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createAgentToken(t *testing.T, client *Client, ap *AgentPool) (*AgentToken, func()) {
	var apCleanup func()

	if ap == nil {
		ap, apCleanup = createAgentPool(t, client, nil)
	}

	ctx := context.Background()
	at, err := client.AgentTokens.Create(ctx, ap.ID, AgentTokenCreateOptions{
		Description: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return at, func() {
		if err := client.AgentTokens.Delete(ctx, at.ID); err != nil {
			t.Errorf("Error destroying agent token! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"AgentToken: %s\nError: %s", at.ID, err)
		}

		if apCleanup != nil {
			apCleanup()
		}
	}
}

func createConfigurationVersion(t *testing.T, client *Client, w *Workspace) (*ConfigurationVersion, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	ctx := context.Background()
	cv, err := client.ConfigurationVersions.Create(
		ctx,
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

	ctx := context.Background()
	err := client.ConfigurationVersions.Upload(ctx, cv.UploadURL, "test-fixtures/config-version")
	if err != nil {
		cvCleanup()
		t.Fatal(err)
	}

	for i := 0; ; i++ {
		cv, err = client.ConfigurationVersions.Read(ctx, cv.ID)
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

func createNotificationConfiguration(t *testing.T, client *Client, w *Workspace, options *NotificationConfigurationCreateOptions) (*NotificationConfiguration, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	if options == nil {
		options = &NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []string{NotificationTriggerCreated},
		}
	}

	ctx := context.Background()
	nc, err := client.NotificationConfigurations.Create(
		ctx,
		w.ID,
		*options,
	)
	if err != nil {
		t.Fatal(err)
	}

	return nc, func() {
		if err := client.NotificationConfigurations.Delete(ctx, nc.ID); err != nil {
			t.Errorf("Error destroying notification configuration! WARNING: Dangling\n"+
				"resources may exist! The full error is shown below.\n\n"+
				"NotificationConfiguration: %s\nError: %s", nc.ID, err)
		}

		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createPolicySetParameter(t *testing.T, client *Client, ps *PolicySet) (*PolicySetParameter, func()) {
	var psCleanup func()

	if ps == nil {
		ps, psCleanup = createPolicySet(t, client, nil, nil, nil)
	}

	ctx := context.Background()
	v, err := client.PolicySetParameters.Create(ctx, ps.ID, PolicySetParameterCreateOptions{
		Key:      String(randomString(t)),
		Value:    String(randomString(t)),
		Category: Category(CategoryPolicySet),
	})
	if err != nil {
		t.Fatal(err)
	}

	return v, func() {
		if err := client.PolicySetParameters.Delete(ctx, ps.ID, v.ID); err != nil {
			t.Errorf("Error destroying variable! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Parameter: %s\nError: %s", v.Key, err)
		}

		if psCleanup != nil {
			psCleanup()
		}
	}
}

func createPolicySet(t *testing.T, client *Client, org *Organization, policies []*Policy, workspaces []*Workspace) (*PolicySet, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	ps, err := client.PolicySets.Create(ctx, org.Name, PolicySetCreateOptions{
		Name:       String(randomString(t)),
		Policies:   policies,
		Workspaces: workspaces,
	})
	if err != nil {
		t.Fatal(err)
	}

	return ps, func() {
		if err := client.PolicySets.Delete(ctx, ps.ID); err != nil {
			t.Errorf("Error destroying policy set! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"PolicySet: %s\nError: %s", ps.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createPolicySetVersion(t *testing.T, client *Client, ps *PolicySet) (*PolicySetVersion, func()) {
	var psCleanup func()

	if ps == nil {
		ps, psCleanup = createPolicySet(t, client, nil, nil, nil)
	}

	ctx := context.Background()
	psv, err := client.PolicySetVersions.Create(ctx, ps.ID)
	if err != nil {
		t.Fatal(err)
	}

	return psv, func() {
		// Deleting a Policy Set Version is done through deleting a Policy Set.
		if psCleanup != nil {
			psCleanup()
		}
	}
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
			{
				Path: String(name + ".sentinel"),
				Mode: EnforcementMode(EnforcementSoft),
			},
		},
	}

	ctx := context.Background()
	p, err := client.Policies.Create(ctx, org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	return p, func() {
		if err := client.Policies.Delete(ctx, p.ID); err != nil {
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

	ctx := context.Background()
	err := client.Policies.Upload(ctx, p.ID, []byte(fmt.Sprintf("main = rule { %t }", pass)))
	if err != nil {
		t.Fatal(err)
	}

	p, err = client.Policies.Read(ctx, p.ID)
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

func createOAuthClient(t *testing.T, client *Client, org *Organization) (*OAuthClient, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		t.Skip("Export a valid GITHUB_TOKEN before running this test!")
	}

	options := OAuthClientCreateOptions{
		APIURL:          String("https://api.github.com"),
		HTTPURL:         String("https://github.com"),
		OAuthToken:      String(githubToken),
		ServiceProvider: ServiceProvider(ServiceProviderGithub),
	}

	ctx := context.Background()
	oc, err := client.OAuthClients.Create(ctx, org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	// This currently panics as the token will not be there when the client is
	// created. To get a token, the client needs to be connected through the UI
	// first. So the test using this (TestOAuthTokensList) is currently disabled.
	return oc, func() {
		if err := client.OAuthClients.Delete(ctx, oc.ID); err != nil {
			t.Errorf("Error destroying OAuth client! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"OAuthClient: %s\nError: %s", oc.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createOAuthToken(t *testing.T, client *Client, org *Organization) (*OAuthToken, func()) {
	ocTest, ocTestCleanup := createOAuthClient(t, client, org)
	return ocTest.OAuthTokens[0], ocTestCleanup
}

func createOrganization(t *testing.T, client *Client) (*Organization, func()) {
	ctx := context.Background()
	org, err := client.Organizations.Create(ctx, OrganizationCreateOptions{
		Name:  String("tst-" + randomString(t)),
		Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
	})
	if err != nil {
		t.Fatal(err)
	}

	return org, func() {
		if err := client.Organizations.Delete(ctx, org.Name); err != nil {
			t.Errorf("Error destroying organization! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Organization: %s\nError: %s", org.Name, err)
		}
	}
}

func createOrganizationMembership(t *testing.T, client *Client, org *Organization) (*OrganizationMembership, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	mem, err := client.OrganizationMemberships.Create(ctx, org.Name, OrganizationMembershipCreateOptions{
		Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
	})
	if err != nil {
		t.Fatal(err)
	}

	return mem, func() {
		if err := client.OrganizationMemberships.Delete(ctx, mem.ID); err != nil {
			t.Errorf("Error destroying membership! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Membership: %s\nError: %s", mem.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createOrganizationToken(t *testing.T, client *Client, org *Organization) (*OrganizationToken, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	tk, err := client.OrganizationTokens.Create(ctx, org.Name)
	if err != nil {
		t.Fatal(err)
	}

	return tk, func() {
		if err := client.OrganizationTokens.Delete(ctx, org.Name); err != nil {
			t.Errorf("Error destroying organization token! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"OrganizationToken: %s\nError: %s", tk.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createRunTrigger(t *testing.T, client *Client, w, sourceable *Workspace) (*RunTrigger, func()) {
	var wCleanup func()
	var sourceableCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	if sourceable == nil {
		sourceable, sourceableCleanup = createWorkspace(t, client, nil)
	}

	ctx := context.Background()
	rt, err := client.RunTriggers.Create(
		ctx,
		w.ID,
		RunTriggerCreateOptions{
			Sourceable: sourceable,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	return rt, func() {
		if err := client.RunTriggers.Delete(ctx, rt.ID); err != nil {
			t.Errorf("Error destroying run trigger! WARNING: Dangling\n"+
				"resources may exist! The full error is shown below.\n\n"+
				"RunTrigger: %s\nError: %s", rt.ID, err)
		}

		if wCleanup != nil {
			wCleanup()
		}

		if sourceableCleanup != nil {
			sourceableCleanup()
		}
	}
}

func createRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	cv, cvCleanup := createUploadedConfigurationVersion(t, client, w)

	ctx := context.Background()
	r, err := client.Runs.Create(ctx, RunCreateOptions{
		ConfigurationVersion: cv,
		Workspace:            w,
	})
	if err != nil {
		t.Fatal(err)
	}

	return r, func() {
		cvCleanup()

		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createRunWithStatus(t *testing.T, client *Client, w *Workspace, timeout int, desiredStatuses ...RunStatus) (*Run, func()) {
	r, rCleanup := createRun(t, client, w)

	var err error
	ctx := context.Background()
	for i := 0; ; i++ {
		r, err = client.Runs.Read(ctx, r.ID)
		if err != nil {
			t.Fatal(err)
		}

		for _, desiredStatus := range desiredStatuses {
			// if we're creating an applied run, we need to manually confirm the apply once the plan finishes
			isApplyable := hasApplyableStatus(r)
			if desiredStatus == RunApplied && isApplyable {
				err := client.Runs.Apply(ctx, r.ID, RunApplyOptions{})
				if err != nil {
					t.Fatal(err)
				}
			} else if desiredStatus == r.Status {
				return r, rCleanup
			}
		}

		if i > timeout {
			runStatus := r.Status
			rCleanup()
			t.Fatal(fmt.Printf("Timeout waiting for run to reach status %v, had status %s", desiredStatuses, runStatus))
		}

		time.Sleep(1 * time.Second)
	}
}

func createPlannedRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	if paidFeaturesDisabled() {
		return createRunWithStatus(t, client, w, 45, RunPlanned)
	}
	return createRunWithStatus(t, client, w, 45, RunCostEstimated)
}

func createCostEstimatedRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	return createRunWithStatus(t, client, w, 45, RunCostEstimated)
}

func createPolicyCheckedRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	return createRunWithStatus(t, client, w, 45, RunPolicyChecked, RunPolicyOverride)
}

func createAppliedRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	return createRunWithStatus(t, client, w, 90, RunApplied)
}

func hasApplyableStatus(r *Run) bool {
	if len(r.PolicyChecks) > 0 {
		return r.Status == RunPolicyChecked || r.Status == RunPolicyOverride
	} else if r.CostEstimate != nil {
		return r.Status == RunCostEstimated
	} else {
		return r.Status == RunPlanned
	}
}

func createPlanExport(t *testing.T, client *Client, r *Run) (*PlanExport, func()) {
	var rCleanup func()

	if r == nil {
		r, rCleanup = createPlannedRun(t, client, nil)
	}

	ctx := context.Background()
	pe, err := client.PlanExports.Create(ctx, PlanExportCreateOptions{
		Plan:     r.Plan,
		DataType: PlanExportType(PlanExportSentinelMockBundleV0),
	})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; ; i++ {
		pe, err := client.PlanExports.Read(ctx, pe.ID)
		if err != nil {
			t.Fatal(err)
		}

		if pe.Status == PlanExportFinished {
			return pe, func() {
				if rCleanup != nil {
					rCleanup()
				}
			}
		}

		if i > 45 {
			rCleanup()
			t.Fatal("Timeout waiting for plan export to finish")
		}

		time.Sleep(1 * time.Second)
	}
}

func createRegistryModule(t *testing.T, client *Client, org *Organization) (*RegistryModule, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()

	options := RegistryModuleCreateOptions{
		Name:     String("name"),
		Provider: String("provider"),
	}
	rm, err := client.RegistryModules.Create(ctx, org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	return rm, func() {
		if err := client.RegistryModules.Delete(ctx, org.Name, rm.Name); err != nil {
			t.Errorf("Error destroying registry module! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Registry Module: %s\nError: %s", rm.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createRegistryModuleWithVersion(t *testing.T, client *Client, org *Organization) (*RegistryModule, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()

	options := RegistryModuleCreateOptions{
		Name:     String("name"),
		Provider: String("provider"),
	}
	rm, err := client.RegistryModules.Create(ctx, org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	optionsModuleVersion := RegistryModuleCreateVersionOptions{
		Version: String("1.0.0"),
	}
	_, err = client.RegistryModules.CreateVersion(ctx, org.Name, rm.Name, rm.Provider, optionsModuleVersion)
	if err != nil {
		t.Fatal(err)
	}

	rm, err = client.RegistryModules.Read(ctx, org.Name, rm.Name, rm.Provider)
	if err != nil {
		t.Fatal(err)
	}

	return rm, func() {
		if err := client.RegistryModules.Delete(ctx, org.Name, rm.Name); err != nil {
			t.Errorf("Error destroying registry module! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Registry Module: %s\nError: %s", rm.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createRunTask(t *testing.T, client *Client, org *Organization) (*RunTask, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	runTaskURL := os.Getenv("TFC_RUN_TASK_URL")
	if runTaskURL == "" {
		t.Error("Cannot create a run task with an empty URL. You must set TFC_RUN_TASK_URL for run task related tests.")
	}

	ctx := context.Background()
	r, err := client.RunTasks.Create(ctx, org.Name, RunTaskCreateOptions{
		Name:     "tst-" + randomString(t),
		URL:      runTaskURL,
		Category: "task",
	})
	if err != nil {
		t.Fatal(err)
	}

	return r, func() {
		if err := client.RunTasks.Delete(ctx, r.ID); err != nil {
			t.Errorf("Error removing Run Task! WARNING: Run task limit\n"+
				"may be reached if not deleted! The full error is shown below.\n\n"+
				"Run Task: %s\nError: %s", r.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createSSHKey(t *testing.T, client *Client, org *Organization) (*SSHKey, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	key, err := client.SSHKeys.Create(ctx, org.Name, SSHKeyCreateOptions{
		Name:  String(randomString(t)),
		Value: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return key, func() {
		if err := client.SSHKeys.Delete(ctx, key.ID); err != nil {
			t.Errorf("Error destroying SSH key! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"SSHKey: %s\nError: %s", key.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createStateVersion(t *testing.T, client *Client, serial int64, w *Workspace) (*StateVersion, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	state, err := ioutil.ReadFile("test-fixtures/state-version/terraform.tfstate")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	_, err = client.Workspaces.Lock(ctx, w.ID, WorkspaceLockOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, err := client.Workspaces.Unlock(ctx, w.ID)
		if err != nil {
			t.Fatal(err)
		}
	}()

	sv, err := client.StateVersions.Create(ctx, w.ID, StateVersionCreateOptions{
		MD5:    String(fmt.Sprintf("%x", md5.Sum(state))),
		Serial: Int64(serial),
		State:  String(base64.StdEncoding.EncodeToString(state)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return sv, func() {
		// There currently isn't a way to delete a state, so we
		// can only cleanup by deleting the workspace.
		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createTeam(t *testing.T, client *Client, org *Organization) (*Team, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	tm, err := client.Teams.Create(ctx, org.Name, TeamCreateOptions{
		Name: String(randomString(t)),
		OrganizationAccess: &OrganizationAccessOptions{
			ManagePolicies:        Bool(true),
			ManagePolicyOverrides: Bool(true),
			ManageProviders:       Bool(true),
			ManageModules:         Bool(true),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	return tm, func() {
		if err := client.Teams.Delete(ctx, tm.ID); err != nil {
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
	var orgCleanup, tmCleanup, wCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	if tm == nil {
		tm, tmCleanup = createTeam(t, client, org)
	}

	if w == nil {
		w, wCleanup = createWorkspace(t, client, org)
	}

	ctx := context.Background()
	ta, err := client.TeamAccess.Add(ctx, TeamAccessAddOptions{
		Access:    Access(AccessAdmin),
		Team:      tm,
		Workspace: w,
	})
	if err != nil {
		t.Fatal(err)
	}

	return ta, func() {
		if err := client.TeamAccess.Remove(ctx, ta.ID); err != nil {
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

		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createTeamToken(t *testing.T, client *Client, tm *Team) (*TeamToken, func()) {
	var tmCleanup func()

	if tm == nil {
		tm, tmCleanup = createTeam(t, client, nil)
	}

	ctx := context.Background()
	tt, err := client.TeamTokens.Create(ctx, tm.ID)
	if err != nil {
		t.Fatal(err)
	}

	return tt, func() {
		if err := client.TeamTokens.Delete(ctx, tm.ID); err != nil {
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

	ctx := context.Background()
	v, err := client.Variables.Create(ctx, w.ID, VariableCreateOptions{
		Key:         String(randomString(t)),
		Value:       String(randomString(t)),
		Category:    Category(CategoryTerraform),
		Description: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return v, func() {
		if err := client.Variables.Delete(ctx, w.ID, v.ID); err != nil {
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

	ctx := context.Background()
	w, err := client.Workspaces.Create(ctx, org.Name, WorkspaceCreateOptions{
		Name: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return w, func() {
		if err := client.Workspaces.Delete(ctx, org.Name, w.Name); err != nil {
			t.Errorf("Error destroying workspace! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Workspace: %s\nError: %s", w.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

// queueAllRuns: Whether runs should be queued immediately after workspace creation. When set to
// false, runs triggered by a VCS change will not be queued until at least one run is manually
// queued. If set to true, a run will be automatically started after the configuration is ingressed
// from VCS.
func createWorkspaceWithVCS(t *testing.T, client *Client, org *Organization, options WorkspaceCreateOptions) (*Workspace, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	oc, ocCleanup := createOAuthToken(t, client, org)

	githubIdentifier := os.Getenv("GITHUB_POLICY_SET_IDENTIFIER")
	if githubIdentifier == "" {
		t.Fatal("Export a valid GITHUB_POLICY_SET_IDENTIFIER before running this test!")
	}

	if options.Name == nil {
		options.Name = String(randomString(t))
	}

	if options.VCSRepo == nil {
		options.VCSRepo = &VCSRepoOptions{
			Identifier:   String(githubIdentifier),
			OAuthTokenID: String(oc.ID),
		}
	}

	ctx := context.Background()
	w, err := client.Workspaces.Create(ctx, org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	return w, func() {
		if err := client.Workspaces.Delete(ctx, org.Name, w.Name); err != nil {
			t.Errorf("Error destroying workspace! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Workspace: %s\nError: %s", w.Name, err)
		}

		if ocCleanup != nil {
			ocCleanup()
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createWorkspaceRunTask(t *testing.T, client *Client, workspace *Workspace, runTask *RunTask) (*WorkspaceRunTask, func()) {
	var organization *Organization
	var runTaskCleanup func()
	var workspaceCleanup func()
	var orgCleanup func()

	if workspace == nil {
		organization, orgCleanup = createOrganization(t, client)
		workspace, workspaceCleanup = createWorkspace(t, client, organization)
	}

	if runTask == nil {
		runTask, runTaskCleanup = createRunTask(t, client, organization)
	}

	ctx := context.Background()
	wr, err := client.WorkspaceRunTasks.Create(ctx, workspace.ID, WorkspaceRunTaskCreateOptions{
		EnforcementLevel: Advisory,
		RunTask:          runTask,
	})
	if err != nil {
		t.Fatal(err)
	}

	return wr, func() {
		if err := client.WorkspaceRunTasks.Delete(ctx, workspace.ID, wr.ID); err != nil {
			t.Errorf("Error destroying workspace run task!\n"+
				"Workspace: %s\n"+
				"Workspace Run Task: %s\n"+
				"Error: %s", workspace.ID, wr.ID, err)
		}

		if runTaskCleanup != nil {
			runTaskCleanup()
		}

		if workspaceCleanup != nil {
			workspaceCleanup()
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func genSha(t *testing.T, secret, data string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, err := h.Write([]byte(data))
	if err != nil {
		t.Fatalf("error writing hmac: %s", err)
	}
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// skips a test if the environment is for Terraform Cloud.
func skipIfCloud(t *testing.T) {
	if !enterpriseEnabled() {
		t.Skip("Skipping test related to Terraform Cloud. Set ENABLE_TFE=1 to run.")
	}
}

// skips a test if the environment is for Terraform Enterprise
func skipIfEnterprise(t *testing.T) {
	if enterpriseEnabled() {
		t.Skip("Skipping test related to Terraform Enterprise. Set ENABLE_TFE=0 to run.")
	}
}

// skips a test if the test requires a paid feature, and this flag
// SKIP_PAID is set.
func skipIfFreeOnly(t *testing.T) {
	if paidFeaturesDisabled() {
		t.Skip("Skipping test that requires a paid feature. Remove 'SKIP_PAID=1' if you want to run this test")
	}
}

// skips a test if the test requires a beta feature
func skipIfBeta(t *testing.T) {
	if !betaFeaturesEnabled() {
		t.Skip("Skipping test related to a Terraform Cloud beta feature. Set ENABLE_BETA=1 to run.")
	}
}

// Checks to see if ENABLE_TFE is set to 1, thereby enabling enterprise tests.
func enterpriseEnabled() bool {
	return os.Getenv("ENABLE_TFE") == "1"
}

func paidFeaturesDisabled() bool {
	return os.Getenv("SKIP_PAID") == "1"
}

// Checks to see if ENABLE_BETA is set to 1, thereby enabling tests for beta features.
func betaFeaturesEnabled() bool {
	return os.Getenv("ENABLE_BETA") == "1"
}
