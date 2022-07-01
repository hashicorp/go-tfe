package tfe

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	uuid "github.com/hashicorp/go-uuid"
)

const badIdentifier = "! / nope" //nolint
const tickDuration = 2

// Memoize test account details
var _testAccountDetails *TestAccountDetails

type featureSet struct {
	ID string `jsonapi:"primary,feature-sets"`
}

type featureSetList struct {
	Items []*featureSet
	*Pagination
}

type featureSetListOptions struct {
	Q string `url:"q,omitempty"`
}

type retryableFn func() (interface{}, error)

type updateFeatureSetOptions struct {
	Type               string    `jsonapi:"primary,subscription"`
	RunsCeiling        int       `jsonapi:"attr,runs-ceiling"`
	ContractStartAt    time.Time `jsonapi:"attr,contract-start-at,iso8601"`
	ContractUserLimit  int       `jsonapi:"attr,contract-user-limit"`
	ContractApplyLimit int       `jsonapi:"attr,contract-apply-limit"`

	FeatureSet *featureSet `jsonapi:"relation,feature-set"`
}

func testClient(t *testing.T) *Client {
	client, err := NewClient(&Config{
		RetryServerErrors: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func testAuditTrailClient(t *testing.T, userClient *Client, org *Organization) *Client {
	upgradeOrganizationSubscription(t, userClient, org)

	orgToken, orgTokenCleanup := createOrganizationToken(t, userClient, org)
	t.Cleanup(orgTokenCleanup)

	client, err := NewClient(&Config{
		Token: orgToken.Token,
	})
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
			t.Logf("Error destroying agent pool! WARNING: Dangling resources "+
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

	WaitUntilStatus(t, client, cv, ConfigurationUploaded, 15)

	return cv, cvCleanup
}

// helper to wait until a configuration version has reached a certain status
func WaitUntilStatus(t *testing.T, client *Client, cv *ConfigurationVersion, desiredStatus ConfigurationStatus, timeoutSeconds int) {
	ctx := context.Background()

	for i := 0; ; i++ {
		refreshed, err := client.ConfigurationVersions.Read(ctx, cv.ID)
		require.NoError(t, err)

		if refreshed.Status == desiredStatus {
			break
		}

		if i > timeoutSeconds {
			t.Fatal("Timeout waiting for the configuration version to be archived")
		}

		time.Sleep(1 * time.Second)
	}
}

func createGPGKey(t *testing.T, client *Client, org *Organization, provider *RegistryProvider) (*GPGKey, func()) {
	var orgCleanup func()
	var providerCleanup func()

	ctx := context.Background()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
		upgradeOrganizationSubscription(t, client, org)
	}

	if provider == nil {
		provider, providerCleanup = createRegistryProvider(t, client, org, PrivateRegistry)
	}

	gpgKey, err := client.GPGKeys.Create(ctx, PrivateRegistry, GPGKeyCreateOptions{
		Namespace:  provider.Organization.Name,
		AsciiArmor: testGpgArmor,
	})
	if err != nil {
		t.Fatal(err)
	}

	return gpgKey, func() {
		if err := client.GPGKeys.Delete(ctx, GPGKeyID{
			RegistryName: PrivateRegistry,
			Namespace:    provider.Organization.Name,
			KeyID:        gpgKey.KeyID,
		}); err != nil {
			t.Errorf("Error removing GPG key! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"GPGKey: %s\nError: %s", gpgKey.KeyID, err)
		}

		if providerCleanup != nil {
			providerCleanup()
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
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
			Triggers:        []NotificationTriggerType{NotificationTriggerCreated},
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

// createOrganization creates an organization for tests using the special prefix
// "tst-" that the API uses especially to grant access to orgs for testing.
// Don't change this prefix unless we refactor the code!
func createOrganization(t *testing.T, client *Client) (*Organization, func()) {
	return createOrganizationWithOptions(t, client, OrganizationCreateOptions{
		Name:  String("tst-" + randomString(t)),
		Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
	})
}

func createOrganizationWithOptions(t *testing.T, client *Client, options OrganizationCreateOptions) (*Organization, func()) {
	ctx := context.Background()
	org, err := client.Organizations.Create(ctx, options)
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
			t.Fatal(fmt.Printf("Timeout waiting for run ID %s to reach status %v, had status %s",
				r.ID,
				desiredStatuses,
				runStatus))
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
		Name:     String(randomString(t)),
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
	rmID := RegistryModuleID{
		Organization: org.Name,
		Name:         rm.Name,
		Provider:     rm.Provider,
	}
	_, err = client.RegistryModules.CreateVersion(ctx, rmID, optionsModuleVersion)
	if err != nil {
		t.Fatal(err)
	}

	rm, err = client.RegistryModules.Read(ctx, rmID)
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

func createRegistryProvider(t *testing.T, client *Client, org *Organization, registryName RegistryName) (*RegistryProvider, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	if (registryName != PublicRegistry) && (registryName != PrivateRegistry) {
		t.Fatal("RegistryName must be public or private")
	}

	ctx := context.Background()

	namespaceName := "test-namespace-" + randomString(t)
	if registryName == PrivateRegistry {
		namespaceName = org.Name
	}

	options := RegistryProviderCreateOptions{
		Name:         "test-registry-provider-" + randomString(t),
		Namespace:    namespaceName,
		RegistryName: registryName,
	}

	prv, err := client.RegistryProviders.Create(ctx, org.Name, options)

	if err != nil {
		t.Fatal(err)
	}

	prv.Organization = org

	return prv, func() {
		id := RegistryProviderID{
			OrganizationName: org.Name,
			RegistryName:     prv.RegistryName,
			Namespace:        prv.Namespace,
			Name:             prv.Name,
		}

		if err := client.RegistryProviders.Delete(ctx, id); err != nil {
			t.Errorf("Error destroying registry provider! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Registry Provider: %s/%s\nError: %s", prv.Namespace, prv.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createRegistryProviderPlatform(t *testing.T, client *Client, provider *RegistryProvider, version *RegistryProviderVersion) (*RegistryProviderPlatform, func()) {
	var providerCleanup func()
	var versionCleanup func()

	if provider == nil {
		provider, providerCleanup = createRegistryProvider(t, client, nil, PrivateRegistry)
	}

	providerID := RegistryProviderID{
		OrganizationName: provider.Organization.Name,
		RegistryName:     provider.RegistryName,
		Namespace:        provider.Namespace,
		Name:             provider.Name,
	}

	if version == nil {
		version, versionCleanup = createRegistryProviderVersion(t, client, provider)
	}

	versionID := RegistryProviderVersionID{
		RegistryProviderID: providerID,
		Version:            version.Version,
	}

	ctx := context.Background()

	options := RegistryProviderPlatformCreateOptions{
		OS:       randomString(t),
		Arch:     randomString(t),
		Shasum:   genSha(t, "secret", "data"),
		Filename: randomString(t),
	}

	rpp, err := client.RegistryProviderPlatforms.Create(ctx, versionID, options)

	if err != nil {
		t.Fatal(err)
	}

	return rpp, func() {
		platformID := RegistryProviderPlatformID{
			RegistryProviderVersionID: versionID,
			OS:                        rpp.OS,
			Arch:                      rpp.Arch,
		}

		if err := client.RegistryProviderPlatforms.Delete(ctx, platformID); err != nil {
			t.Errorf("Error destroying registry provider platform! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Registry Provider Version: %s/%s/%s/%s\nError: %s", rpp.RegistryProviderVersion.RegistryProvider.Namespace, rpp.RegistryProviderVersion.RegistryProvider.Name, rpp.OS, rpp.Arch, err)
		}

		if versionCleanup != nil {
			versionCleanup()
		}
		if providerCleanup != nil {
			providerCleanup()
		}
	}
}

func createRegistryProviderVersion(t *testing.T, client *Client, provider *RegistryProvider) (*RegistryProviderVersion, func()) {
	var providerCleanup func()

	if provider == nil {
		provider, providerCleanup = createRegistryProvider(t, client, nil, PrivateRegistry)
	}

	providerID := RegistryProviderID{
		OrganizationName: provider.Organization.Name,
		RegistryName:     provider.RegistryName,
		Namespace:        provider.Namespace,
		Name:             provider.Name,
	}

	ctx := context.Background()

	options := RegistryProviderVersionCreateOptions{
		Version:   randomSemver(t),
		KeyID:     randomString(t),
		Protocols: []string{"4.0", "5.0", "6.0"},
	}

	prvv, err := client.RegistryProviderVersions.Create(ctx, providerID, options)

	if err != nil {
		t.Fatal(err)
	}

	prvv.RegistryProvider = provider

	return prvv, func() {
		id := RegistryProviderVersionID{
			Version:            options.Version,
			RegistryProviderID: providerID,
		}

		if err := client.RegistryProviderVersions.Delete(ctx, id); err != nil {
			t.Errorf("Error destroying registry provider version! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Registry Provider Version: %s/%s/%s\nError: %s", prvv.RegistryProvider.Namespace, prvv.RegistryProvider.Name, prvv.Version, err)
		}

		if providerCleanup != nil {
			providerCleanup()
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
	return createWorkspaceWithOptions(t, client, org, WorkspaceCreateOptions{
		Name: String(randomString(t)),
	})
}

func createWorkspaceWithOptions(t *testing.T, client *Client, org *Organization, options WorkspaceCreateOptions) (*Workspace, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
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

func createVariableSet(t *testing.T, client *Client, org *Organization, options VariableSetCreateOptions) (*VariableSet, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	if options.Name == nil {
		options.Name = String(randomString(t))
	}

	if options.Global == nil {
		options.Global = Bool(false)
	}

	ctx := context.Background()
	vs, err := client.VariableSets.Create(ctx, org.Name, &options)
	if err != nil {
		t.Fatal(err)
	}

	return vs, func() {
		if err := client.VariableSets.Delete(ctx, vs.ID); err != nil {
			t.Errorf("Error destroying variable set! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"VariableSet: %s\nError: %s", vs.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createVariableSetVariable(t *testing.T, client *Client, vs *VariableSet, options VariableSetVariableCreateOptions) (*VariableSetVariable, func()) {
	var vsCleanup func()

	if vs == nil {
		vs, vsCleanup = createVariableSet(t, client, nil, VariableSetCreateOptions{})
	}

	if options.Key == nil {
		options.Key = String(randomString(t))
	}

	if options.Value == nil {
		options.Value = String(randomString(t))
	}

	if options.Description == nil {
		options.Description = String("")
	}

	if options.Category == nil {
		options.Category = Category(CategoryTerraform)
	}

	if options.HCL == nil {
		options.HCL = Bool(false)
	}

	if options.Sensitive == nil {
		options.Sensitive = Bool(false)
	}

	ctx := context.Background()
	v, err := client.VariableSetVariables.Create(ctx, vs.ID, &options)
	if err != nil {
		t.Fatal(err)
	}

	return v, func() {
		if err := client.VariableSetVariables.Delete(ctx, vs.ID, v.ID); err != nil {
			t.Errorf("Error destroying variable! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Variable: %s\nError: %s", v.Key, err)
		}

		if vsCleanup != nil {
			vsCleanup()
		}
	}
}

// Attempts to upgrade an organization to the business plan. Requires a user token with admin access.
func upgradeOrganizationSubscription(t *testing.T, client *Client, organization *Organization) {
	if enterpriseEnabled() {
		t.Skip("Can not upgrade an organization's subscription when enterprise is enabled. Set ENABLE_TFE=0 to run.")
	}

	req, err := client.newRequest("GET", "admin/feature-sets", featureSetListOptions{
		Q: "Business",
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	fsl := &featureSetList{}
	err = client.do(context.Background(), req, fsl)
	if err != nil {
		t.Fatalf("failed to enumerate feature sets: %v", err)
		return
	} else if len(fsl.Items) == 0 {
		t.Fatalf("feature set response was empty")
		return
	}

	opts := updateFeatureSetOptions{
		RunsCeiling:        10,
		ContractStartAt:    time.Now(),
		ContractUserLimit:  1000,
		ContractApplyLimit: 5000,
		FeatureSet:         fsl.Items[0],
	}

	u := fmt.Sprintf("admin/organizations/%s/subscription", url.QueryEscape(organization.Name))
	req, err = client.newRequest("POST", u, &opts)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
		return
	}

	err = client.do(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("Failed to upgrade subscription: %v", err)
	}
}

func waitForSVOutputs(t *testing.T, client *Client, svID string) {
	t.Helper()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		_, err := retry(func() (interface{}, error) {
			outputs, err := client.StateVersions.ListOutputs(context.Background(), svID, nil)
			if err != nil {
				return nil, err
			}

			if len(outputs.Items) == 0 {
				return nil, errors.New("no state version outputs found")
			}

			return outputs, nil
		})
		if err != nil {
			t.Error(err)
		}

		wg.Done()
	}()

	wg.Wait()
}

func waitForRunLock(t *testing.T, client *Client, workspaceID string) {
	t.Helper()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		_, err := retry(func() (interface{}, error) {
			ws, err := client.Workspaces.ReadByID(context.Background(), workspaceID)
			if err != nil {
				return nil, err
			}

			if !ws.Locked {
				return nil, errors.New("workspace is not locked by run")
			}

			return ws, nil
		})
		if err != nil {
			t.Error(err)
		}

		wg.Done()
	}()

	wg.Wait()
}

func retry(f retryableFn) (interface{}, error) { //nolint
	tick := time.NewTicker(tickDuration * time.Second)
	retries := 0
	maxRetries := 5

	defer tick.Stop()

	for { //nolint
		select {
		case <-tick.C:
			res, err := f()
			if err == nil {
				return res, nil
			}

			if retries >= maxRetries {
				return nil, err
			}

			retries += 1
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

// genSafeRandomTerraformVersion returns a random version number of the form
// `1.0.<RANDOM>`, which TFC won't ever select as the latest available
// Terraform. (At the time of writing, a fresh TFC instance will include
// official Terraforms 1.2 and higher.) This is necessary because newly created
// workspaces default to the latest available version, and there's nothing
// preventing unrelated processes from creating workspaces during these tests.
func genSafeRandomTerraformVersion() string {
	rInt := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	// Avoid colliding with an official Terraform version. Highest 1.0 was
	// 1.0.11, so add a little padding and call it good.
	for rInt < 20 {
		rInt = rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	}
	return fmt.Sprintf("1.0.%d", rInt)
}

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func randomSemver(t *testing.T) string {
	return fmt.Sprintf("%d.%d.%d", rand.Intn(99)+3, rand.Intn(99)+1, rand.Intn(99)+1)
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

// Useless key but enough to pass validation in the API
const testGpgArmor string = `
-----BEGIN PGP PUBLIC KEY BLOCK-----

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
=bBSS
-----END PGP PUBLIC KEY BLOCK-----
`
