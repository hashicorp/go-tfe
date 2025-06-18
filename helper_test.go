// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	uuid "github.com/hashicorp/go-uuid"
)

const badIdentifier = "! / nope" //nolint
const agentVersion = "1.3.0"
const testInitialClientToken = "insert-your-token-here"
const testTaskResultCallbackToken = "this-is-task-result-callback-token"

var _testAccountDetails *TestAccountDetails

func testClient(t *testing.T) *Client {
	client, err := NewClient(&Config{
		RetryServerErrors: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	return client
}

type adminRoleType string

const (
	siteAdmin                adminRoleType = "site-admin"
	configurationAdmin       adminRoleType = "configuration"
	provisionLicensesAdmin   adminRoleType = "provision-licenses"
	subscriptionAdmin        adminRoleType = "subscription"
	supportAdmin             adminRoleType = "support"
	securityMaintenanceAdmin adminRoleType = "security-maintenance"
	versionMaintenanceAdmin  adminRoleType = "version-maintenance"
)

func getTokenForAdminRole(adminRole adminRoleType) string {
	token := ""

	switch adminRole {
	case siteAdmin:
		token = os.Getenv("TFE_ADMIN_SITE_ADMIN_TOKEN")
	case configurationAdmin:
		token = os.Getenv("TFE_ADMIN_CONFIGURATION_TOKEN")
	case provisionLicensesAdmin:
		token = os.Getenv("TFE_ADMIN_PROVISION_LICENSES_TOKEN")
	case subscriptionAdmin:
		token = os.Getenv("TFE_ADMIN_SUBSCRIPTION_TOKEN")
	case supportAdmin:
		token = os.Getenv("TFE_ADMIN_SUPPORT_TOKEN")
	case securityMaintenanceAdmin:
		token = os.Getenv("TFE_ADMIN_SECURITY_MAINTENANCE_TOKEN")
	case versionMaintenanceAdmin:
		token = os.Getenv("TFE_ADMIN_VERSION_MAINTENANCE_TOKEN")
	}

	return token
}

func testAdminClient(t *testing.T, adminRole adminRoleType) *Client {
	token := getTokenForAdminRole(adminRole)
	if token == "" {
		t.Fatal("missing API token for admin role " + adminRole)
	}

	client, err := NewClient(&Config{
		Token:             token,
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

// TestAccountDetails represents the basic account information
// of a Terraform Enterprise or HCP Terraform user.
//
// See FetchTestAccountDetails for more information.
type TestAccountDetails struct {
	ID       string `jsonapi:"primary,users"`
	Username string `jsonapi:"attr,username"`
	Email    string `jsonapi:"attr,email"`
}

func fetchTestAccountDetails(t *testing.T, client *Client) *TestAccountDetails {
	t.Helper()

	if _testAccountDetails == nil {
		_testAccountDetails = &TestAccountDetails{}
		req, err := client.NewRequest("GET", "account/details", nil)
		if err != nil {
			t.Fatalf("could not create account details request: %v", err)
		}

		ctx := context.Background()
		err = req.Do(ctx, _testAccountDetails)
		if err != nil {
			t.Fatalf("could not fetch test user details: %v", err)
		}
	}

	return _testAccountDetails
}

func downloadFile(filePath, fileURL string) error {
	// Get the data
	resp, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(zf *zip.File) error {
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, zf.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if zf.FileInfo().IsDir() {
			return os.MkdirAll(path, zf.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(path), zf.Mode()); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zf.Mode())
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()

		_, err = io.Copy(f, rc)
		if err != nil {
			return err
		}

		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadTFCAgent(t *testing.T) (string, error) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "tfc-agent")
	if err != nil {
		return "", fmt.Errorf("cannot create temp dir: %w", err)
	}
	t.Cleanup(func() {
		fmt.Printf("cleaning up %s \n", tmpDir)
		os.RemoveAll(tmpDir)
	})
	agentPath := fmt.Sprintf("https://releases.hashicorp.com/tfc-agent/%s/tfc-agent_%s_linux_amd64.zip", agentVersion, agentVersion)
	zipFile := fmt.Sprintf("%s/agent.zip", tmpDir)

	if err = downloadFile(zipFile, agentPath); err != nil {
		return "", fmt.Errorf("cannot download agent file: %w", err)
	}

	if err = unzip(zipFile, tmpDir); err != nil {
		return "", fmt.Errorf("cannot unzip file: %w", err)
	}
	return fmt.Sprintf("%s/tfc-agent", tmpDir), nil
}

func createAgent(t *testing.T, client *Client, org *Organization) (*Agent, *AgentPool, func()) {
	var orgCleanup func()
	var agentPoolTokenCleanup func()
	var agent *Agent
	var ok bool

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	agentPool, agentPoolCleanup := createAgentPool(t, client, org)

	upgradeOrganizationSubscription(t, client, org)

	agentPoolToken, agentPoolTokenCleanup := createAgentToken(t, client, agentPool)

	cleanup := func() {
		agentPoolTokenCleanup()

		if agentPoolCleanup != nil {
			agentPoolCleanup()
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}

	agentPath, err := downloadTFCAgent(t)
	if err != nil {
		return agent, agentPool, cleanup
	}

	ctx := context.Background()

	cmd := exec.Command(agentPath)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"TFC_AGENT_TOKEN="+agentPoolToken.Token,
		"TFC_AGENT_NAME="+"test-agent",
		"TFC_ADDRESS="+DefaultConfig().Address,
	)

	go func() {
		_, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Could not run container: %s", err)
		}
	}()

	t.Cleanup(func() {
		if err := cmd.Process.Kill(); err != nil {
			t.Error(err)
		}
	})

	i, err := retry(func() (interface{}, error) {
		agentList, err := client.Agents.List(ctx, agentPool.ID, nil)
		if err != nil {
			return nil, err
		}

		if agentList != nil && len(agentList.Items) > 0 {
			return agentList.Items[0], nil
		}
		return nil, errors.New("no agent found")
	})

	if err != nil {
		t.Fatalf("Could not return an agent %s", err)
	}

	agent, ok = i.(*Agent)
	if !ok {
		t.Fatalf("Expected type to be *Agent but got %T", agent)
	}

	return agent, agentPool, cleanup
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

func createAgentPoolWithOptions(t *testing.T, client *Client, org *Organization, opts AgentPoolCreateOptions) (*AgentPool, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	pool, err := client.AgentPools.Create(ctx, org.Name, opts)
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

func createTestRunConfigurationVersion(t *testing.T, client *Client, rm *RegistryModule) (*ConfigurationVersion, func()) {
	var rmCleanup func()

	if rm == nil {
		rm, rmCleanup = createRegistryModuleWithVersion(t, client, nil)
	}

	ctx := context.Background()
	cv, err := client.ConfigurationVersions.CreateForRegistryModule(
		ctx,
		RegistryModuleID{
			Organization: rm.Organization.Name,
			Name:         rm.Name,
			Provider:     rm.Provider,
			Namespace:    rm.Namespace,
			RegistryName: rm.RegistryName,
		})
	if err != nil {
		t.Fatal(err)
	}

	return cv, func() {
		if rmCleanup != nil {
			rmCleanup()
		}
	}
}

func createUploadedTestRunConfigurationVersion(t *testing.T, client *Client, rm *RegistryModule) (*ConfigurationVersion, func()) {
	cv, cvCleanup := createTestRunConfigurationVersion(t, client, rm)

	ctx := context.Background()
	err := client.ConfigurationVersions.Upload(ctx, cv.UploadURL, "test-fixtures/config-version-with-test")
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

	runTaskURL := os.Getenv("TFC_RUN_TASK_URL")
	if runTaskURL == "" {
		t.Skip("Cannot create a notification configuration with an empty URL. You must set TFC_RUN_TASK_URL for run task related tests.")
	}

	if options == nil {
		options = &NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			URL:             String(runTaskURL),
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

func createTeamNotificationConfiguration(t *testing.T, client *Client, team *Team, options *NotificationConfigurationCreateOptions) (*NotificationConfiguration, func()) {
	var tCleanup func()

	if team == nil {
		team, tCleanup = createTeam(t, client, nil)
	}

	// Team notification configurations do not actually require a run task, but we'll
	// reuse this as a URL that returns a 200.
	runTaskURL := os.Getenv("TFC_RUN_TASK_URL")
	if runTaskURL == "" {
		t.Error("You must set TFC_RUN_TASK_URL for run task related tests.")
	}

	if options == nil {
		options = &NotificationConfigurationCreateOptions{
			DestinationType:    NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:            Bool(false),
			Name:               String(randomString(t)),
			Token:              String(randomString(t)),
			URL:                String(runTaskURL),
			Triggers:           []NotificationTriggerType{NotificationTriggerChangeRequestCreated},
			SubscribableChoice: &NotificationConfigurationSubscribableChoice{Team: team},
		}
	}

	ctx := context.Background()
	nc, err := client.NotificationConfigurations.Create(
		ctx,
		team.ID,
		*options,
	)
	if err != nil {
		t.Fatal(err)
	}

	return nc, func() {
		if err := client.NotificationConfigurations.Delete(ctx, nc.ID); err != nil {
			t.Errorf("Error destroying team notification configuration! WARNING: Dangling\n"+
				"resources may exist! The full error is shown below.\n\n"+
				"NotificationConfiguration: %s\nError: %s", nc.ID, err)
		}

		if tCleanup != nil {
			tCleanup()
		}
	}
}

func createPolicySetParameter(t *testing.T, client *Client, ps *PolicySet) (*PolicySetParameter, func()) {
	var psCleanup func()

	if ps == nil {
		ps, psCleanup = createPolicySet(t, client, nil, nil, nil, nil, nil, "")
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

func createPolicySet(t *testing.T, client *Client, org *Organization, policies []*Policy, workspaces []*Workspace,
	excludedWorkspace []*Workspace, projects []*Project, kind PolicyKind) (*PolicySet, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	ps, err := client.PolicySets.Create(ctx, org.Name, PolicySetCreateOptions{
		Name:                String(randomString(t)),
		Policies:            policies,
		Workspaces:          workspaces,
		WorkspaceExclusions: excludedWorkspace,
		Projects:            projects,
		Kind:                kind,
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

func createPolicySetWithOptions(t *testing.T, client *Client, org *Organization, policies []*Policy, workspaces, excludedWorkspace []*Workspace, projects []*Project, opts PolicySetCreateOptions) (*PolicySet, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	ps, err := client.PolicySets.Create(ctx, org.Name, PolicySetCreateOptions{
		Name:                String(randomString(t)),
		Policies:            policies,
		Workspaces:          workspaces,
		WorkspaceExclusions: excludedWorkspace,
		Projects:            projects,
		Kind:                opts.Kind,
		Overridable:         opts.Overridable,
		AgentEnabled:        opts.AgentEnabled,
		PolicyToolVersion:   opts.PolicyToolVersion,
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
		ps, psCleanup = createPolicySet(t, client, nil, nil, nil, nil, nil, "")
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

func createPolicyWithOptions(t *testing.T, client *Client, org *Organization, opts PolicyCreateOptions) (*Policy, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	name := randomString(t)
	options := PolicyCreateOptions{
		Name:             String(name),
		Kind:             opts.Kind,
		Query:            opts.Query,
		EnforcementLevel: opts.EnforcementLevel,
	}

	if len(opts.Enforce) > 0 {
		path := name + ".sentinel"
		if opts.Kind == OPA {
			path = name + ".rego"
		}
		options.Enforce = []*EnforcementOptions{
			{
				Path: String(path),
				Mode: opts.Enforce[0].Mode,
			},
		}
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

func createUploadedPolicyWithOptions(t *testing.T, client *Client, pass bool, org *Organization, opts PolicyCreateOptions) (*Policy, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	p, pCleanup := createPolicyWithOptions(t, client, org, opts)

	ctx := context.Background()
	policy := fmt.Sprintf("main = rule { %t }", pass)
	if opts.Kind == OPA {
		policy = `package example rule["not allowed"] { false }`
		if !pass {
			policy = `package example rule["not allowed"] { true }`
		}
	}
	err := client.Policies.Upload(ctx, p.ID, []byte(policy))
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

func createOAuthClient(t *testing.T, client *Client, org *Organization, projects []*Project) (*OAuthClient, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	githubToken := os.Getenv("OAUTH_CLIENT_GITHUB_TOKEN")
	if githubToken == "" {
		t.Skip("Export a valid OAUTH_CLIENT_GITHUB_TOKEN before running this test!")
	}

	options := OAuthClientCreateOptions{
		APIURL:          String("https://api.github.com"),
		HTTPURL:         String("https://github.com"),
		OAuthToken:      String(githubToken),
		ServiceProvider: ServiceProvider(ServiceProviderGithub),
		Projects:        projects,
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
	ocTest, ocTestCleanup := createOAuthClient(t, client, org, nil)
	return ocTest.OAuthTokens[0], ocTestCleanup
}

// createOrganization creates an organization for tests using the special prefix
// "tst-" that the API uses especially to grant access to orgs for testing.
// Don't change this prefix unless we refactor the code!
func createOrganization(t *testing.T, client *Client) (*Organization, func()) {
	return createOrganizationWithOptions(t, client, OrganizationCreateOptions{
		Name:                  String("tst-" + randomString(t)),
		Email:                 String(fmt.Sprintf("%s@tfe.local", randomString(t))),
		CostEstimationEnabled: Bool(true),
		StacksEnabled:         Bool(true),
	})
}

func createOrganizationWithOptions(t *testing.T, client *Client, options OrganizationCreateOptions) (*Organization, func()) {
	ctx := context.Background()
	org, err := client.Organizations.Create(ctx, options)
	if err != nil {
		t.Fatalf("Failed to create organization: %s", err)
	}

	return org, func() {
		if err := client.Organizations.Delete(ctx, org.Name); err != nil {
			t.Logf("Error destroying organization! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Organization: %s\nError: %s", org.Name, err)
		}
	}
}

func createOrganizationWithDefaultAgentPool(t *testing.T, client *Client) (*Organization, func()) {
	ctx := context.Background()
	org, orgCleanup := createOrganizationWithOptions(t, client, OrganizationCreateOptions{
		Name:                  String("tst-" + randomString(t)),
		Email:                 String(fmt.Sprintf("%s@tfe.local", randomString(t))),
		CostEstimationEnabled: Bool(true),
	})

	agentPool, _ := createAgentPool(t, client, org)

	org, err := client.Organizations.Update(ctx, org.Name, OrganizationUpdateOptions{
		DefaultExecutionMode: String("agent"),
		DefaultAgentPool:     agentPool,
	})

	if err != nil {
		t.Fatal(err)
	}

	return org, func() {
		// delete the org
		orgCleanup()
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

func createOrganizationTokenWithOptions(t *testing.T, client *Client, org *Organization, options OrganizationTokenCreateOptions) (*OrganizationToken, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	tk, err := client.OrganizationTokens.CreateWithOptions(ctx, org.Name, options)
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

func createPolicyCheckedRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	return createRunWaitForAnyStatuses(t, client, w, []RunStatus{RunPolicyChecked, RunPolicyOverride})
}

func createPlannedRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	return createRunWaitForAnyStatuses(t, client, w, []RunStatus{RunCostEstimated, RunPlanned})
}

func createCostEstimatedRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	return createRunWaitForStatus(t, client, w, RunCostEstimated)
}

func createRunApply(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	ctx := context.Background()
	run, rCleanup := createRunUnapplied(t, client, w)
	timeout := 2 * time.Minute

	// If the run was not in error, it must be applyable
	applyRun(t, client, ctx, run)

	ctxPollRunApplied, cancelPollApplied := context.WithTimeout(ctx, timeout)

	run = pollRunStatus(t, client, ctxPollRunApplied, run, []RunStatus{RunApplied, RunErrored})
	if run.Status == RunErrored {
		fatalDumpRunLog(t, client, ctx, run)
	}

	return run, func() {
		rCleanup()
		cancelPollApplied()
	}
}

func createRunUnapplied(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	var rCleanup func()
	ctx := context.Background()
	r, rCleanup := createRun(t, client, w)

	timeout := 2 * time.Minute

	ctxPollRunReady, cancelPollRunReady := context.WithTimeout(ctx, timeout)

	run := pollRunStatus(
		t,
		client,
		ctxPollRunReady,
		r,
		append(applyableStatuses(r), RunErrored),
	)

	if run.Status == RunErrored {
		fatalDumpRunLog(t, client, ctx, run)
	}

	return run, func() {
		rCleanup()
		cancelPollRunReady()
	}
}

func createRunWaitForStatus(t *testing.T, client *Client, w *Workspace, status RunStatus) (*Run, func()) {
	return createRunWaitForAnyStatuses(t, client, w, []RunStatus{status})
}

func createRunWaitForAnyStatuses(t *testing.T, client *Client, w *Workspace, statuses []RunStatus) (*Run, func()) {
	var rCleanup func()
	ctx := context.Background()
	r, rCleanup := createRun(t, client, w)

	timeout := 2 * time.Minute

	ctxPollRunReady, cancelPollRunReady := context.WithTimeout(ctx, timeout)

	run := pollRunStatus(
		t,
		client,
		ctxPollRunReady,
		r,
		append(statuses, RunErrored),
	)

	if run.Status == RunErrored {
		fatalDumpRunLog(t, client, ctx, run)
	}

	return run, func() {
		rCleanup()
		cancelPollRunReady()
	}
}

func applyableStatuses(r *Run) []RunStatus {
	if len(r.PolicyChecks) > 0 {
		return []RunStatus{
			RunPolicyChecked,
			RunPolicyOverride,
		}
	} else if r.CostEstimate != nil {
		return []RunStatus{RunCostEstimated}
	}

	return []RunStatus{RunPlanned}
}

// pollRunStatus will poll the given run until its status matches one of the given run statuses or the given context
// times out.
func pollRunStatus(t *testing.T, client *Client, ctx context.Context, r *Run, rss []RunStatus) *Run {
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Logf("No deadline was set to poll run %q which could result in an infinite loop", r.ID)
	}

	t.Logf("Polling run %q for status included in %q with deadline of %s", r.ID, rss, deadline)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for finished := false; !finished; {
		t.Log("...")
		select {
		case <-ctx.Done():
			t.Fatalf("Run %q had status %q at deadline", r.ID, r.Status)
		case <-ticker.C:
			r = readRun(t, client, ctx, r)
			t.Logf("Run %q had status %q", r.ID, r.Status)
			for _, rs := range rss {
				if rs == r.Status {
					finished = true
					break
				}
			}
		}
	}

	return r
}

// pollStateVersionStatus will poll the given state version until its status
// matches one of the given statuses or the given context times out.
func pollStateVersionStatus(t *testing.T, client *Client, ctx context.Context, sv *StateVersion, statuses []StateVersionStatus) *StateVersion {
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Logf("No deadline was set to poll state version %q which could result in an infinite loop", sv.ID)
	}

	t.Logf("Polling state version %q for status included in %q with deadline of %s", sv.ID, statuses, deadline)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var err error

	for finished := false; !finished; {
		t.Log("...")
		select {
		case <-ctx.Done():
			t.Fatalf("State version %q had status %q at deadline", sv.ID, sv.Status)
		case <-ticker.C:
			sv, err = client.StateVersions.Read(ctx, sv.ID)
			if err != nil {
				t.Fatalf("Could not read state version %q: %s", sv.ID, err)
			}
			t.Logf("State version %q had status %q", sv.ID, sv.Status)
			for _, svst := range statuses {
				if svst == sv.Status {
					finished = true
					break
				}
			}
		}
	}

	return sv
}

// readRun will re-read the given run.
func readRun(t *testing.T, client *Client, ctx context.Context, r *Run) *Run {
	t.Logf("Reading run %q", r.ID)

	rr, err := client.Runs.Read(ctx, r.ID)
	if err != nil {
		t.Fatalf("Could not read run %q: %s", r.ID, err)
	}

	return rr
}

// applyRun will apply the given run.
func applyRun(t *testing.T, client *Client, ctx context.Context, r *Run) {
	t.Logf("Applying run %q", r.ID)

	if err := client.Runs.Apply(ctx, r.ID, RunApplyOptions{}); err != nil {
		t.Fatalf("Could not apply run %q: %s", r.ID, err)
	}
}

// readPlan will read the given plan.
func readPlan(t *testing.T, client *Client, ctx context.Context, p *Plan) *Plan {
	t.Logf("Reading plan %q", p.ID)

	rp, err := client.Plans.Read(ctx, p.ID)
	if err != nil {
		t.Fatalf("Could not read plan %q: %s", p.ID, err)
	}

	return rp
}

// readPlanLogs will read the logs of the given plan.
func readPlanLogs(t *testing.T, client *Client, ctx context.Context, p *Plan) io.Reader {
	t.Logf("Reading logs of plan %q", p.ID)

	r, err := client.Plans.Logs(ctx, p.ID)
	if err != nil {
		t.Fatalf("Could not retrieve logs of plan %q: %s", p.ID, err)
	}

	return r
}

func fatalDumpRunLog(t *testing.T, client *Client, ctx context.Context, run *Run) {
	t.Helper()
	p := readPlan(t, client, ctx, run.Plan)
	r := readPlanLogs(t, client, ctx, p)

	l, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Could not read logs of plan %q: %v", p.ID, err)
	}

	t.Log("Run errored - here's some logs to help figure out what happened")
	t.Logf("---Start of logs---\n%s\n---End of logs---", l)

	t.Fatalf("Run %q unexpectedly errored", run.ID)
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

func createTestRun(t *testing.T, client *Client, rm *RegistryModule, variables ...*RunVariable) (*TestRun, func()) {
	var rmCleanup func()

	if rm == nil {
		rm, rmCleanup = createBranchBasedRegistryModule(t, client, nil)
	}

	cv, cvCleanup := createUploadedTestRunConfigurationVersion(t, client, rm)

	ctx := context.Background()
	tr, err := client.TestRuns.Create(ctx, TestRunCreateOptions{
		Variables:            variables,
		ConfigurationVersion: cv,
		RegistryModule:       rm,
	})
	if err != nil {
		t.Fatal(err)
	}

	return tr, func() {
		cvCleanup()

		if rmCleanup != nil {
			rmCleanup()
		}
	}
}

func createTestVariable(t *testing.T, client *Client, rm *RegistryModule) (*Variable, func()) {
	var rmCleanup func()

	if rm == nil {
		rm, rmCleanup = createBranchBasedRegistryModule(t, client, nil)
	}
	rmID := RegistryModuleID{
		Organization: rm.Organization.Name,
		Name:         rm.Name,
		Provider:     rm.Provider,
		Namespace:    rm.Namespace,
		RegistryName: rm.RegistryName,
	}

	ctx := context.Background()
	v, err := client.TestVariables.Create(ctx, rmID, VariableCreateOptions{
		Key:         String(randomKeyValue(t)),
		Value:       String(randomStringWithoutSpecialChar(t)),
		Category:    Category(CategoryEnv),
		Description: String(randomStringWithoutSpecialChar(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return v, func() {
		if err := client.TestVariables.Delete(ctx, rmID, v.ID); err != nil {
			t.Errorf("Error destroying variable! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Variable: %s\nError: %s", v.Key, err)
		}

		if rmCleanup != nil {
			rmCleanup()
		}
	}
}

// helper to wait until a test run has reached a certain status
func waitUntilTestRunStatus(t *testing.T, client *Client, rm RegistryModuleID, tr *TestRun, desiredStatus TestRunStatus, timeoutSeconds int) {
	ctx := context.Background()

	for i := 0; ; i++ {
		refreshed, err := client.TestRuns.Read(ctx, rm, tr.ID)
		require.NoError(t, err)

		if refreshed.Status == desiredStatus {
			break
		}

		if i > timeoutSeconds {
			t.Fatalf("Timeout waiting for the test run status %s", string(desiredStatus))
		}

		time.Sleep(1 * time.Second)
	}
}

func createPlanExport(t *testing.T, client *Client, r *Run) (*PlanExport, func()) {
	var rCleanup func()

	if r == nil {
		r, rCleanup = createRunApply(t, client, nil)
	}

	ctx := context.Background()
	pe, err := client.PlanExports.Create(ctx, PlanExportCreateOptions{
		Plan:     r.Plan,
		DataType: PlanExportType(PlanExportSentinelMockBundleV0),
	})
	if err != nil {
		t.Fatal(err)
	}

	timeout := 10 * time.Minute

	ctxPollExportReady, cancelPollExportReady := context.WithTimeout(ctx, timeout)
	t.Cleanup(cancelPollExportReady)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		t.Log("...")
		select {
		case <-ctxPollExportReady.Done():
			rCleanup()
			t.Fatalf("Run %q had status %q at deadline", r.ID, r.Status)
		case <-ticker.C:
			pe, err := client.PlanExports.Read(ctxPollExportReady, pe.ID)
			if err != nil {
				t.Fatal(err)
			}

			if pe.Status == PlanExportFinished || pe.Status == PlanExportQueued {
				return pe, func() {
					if rCleanup != nil {
						rCleanup()
					}
				}
			} else if pe.Status == PlanExportErrored {
				t.Fatal("Plan export failed")
			} else {
				t.Logf("Waiting for plan export finished or queued but was %s", pe.Status)
			}
		}
	}
}

func createBranchBasedRegistryModule(t *testing.T, client *Client, org *Organization) (*RegistryModule, func()) {
	githubIdentifier := os.Getenv("GITHUB_REGISTRY_MODULE_IDENTIFIER")
	if githubIdentifier == "" {
		t.Skip("Export a valid GITHUB_REGISTRY_MODULE_IDENTIFIER before running this test")
	}

	githubBranch := os.Getenv("GITHUB_REGISTRY_MODULE_BRANCH")
	if githubBranch == "" {
		githubBranch = "main"
	}

	var orgCleanup func()
	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	oauthTokenTest, oauthTokenTestCleanup := createOAuthToken(t, client, org)

	ctx := context.Background()

	rm, err := client.RegistryModules.CreateWithVCSConnection(ctx, RegistryModuleCreateWithVCSConnectionOptions{
		VCSRepo: &RegistryModuleVCSRepoOptions{
			OrganizationName:  String(org.Name),
			Identifier:        String(githubIdentifier),
			OAuthTokenID:      String(oauthTokenTest.ID),
			DisplayIdentifier: String(githubIdentifier),
			Branch:            String(githubBranch),
		},
		InitialVersion: String("1.0.0"),
	})

	if err != nil {
		oauthTokenTestCleanup()

		if orgCleanup != nil {
			orgCleanup()
		}

		t.Fatal(err)
	}

	return rm, func() {
		if err := client.RegistryModules.Delete(ctx, org.Name, rm.Name); err != nil {
			t.Errorf("Error destroying registry module! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Registry Module: %s\nError: %s", rm.Name, err)
		}

		oauthTokenTestCleanup()

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createBranchBasedRegistryModuleWithTests(t *testing.T, client *Client, org *Organization) (*RegistryModule, func()) {
	githubIdentifier := os.Getenv("GITHUB_REGISTRY_MODULE_IDENTIFIER")
	if githubIdentifier == "" {
		t.Skip("Export a valid GITHUB_REGISTRY_MODULE_IDENTIFIER before running this test")
	}

	githubBranch := os.Getenv("GITHUB_REGISTRY_MODULE_BRANCH")
	if githubBranch == "" {
		githubBranch = "main"
	}

	var orgCleanup func()
	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	oauthTokenTest, oauthTokenTestCleanup := createOAuthToken(t, client, org)

	ctx := context.Background()

	rm, err := client.RegistryModules.CreateWithVCSConnection(ctx, RegistryModuleCreateWithVCSConnectionOptions{
		VCSRepo: &RegistryModuleVCSRepoOptions{
			OrganizationName:  String(org.Name),
			Identifier:        String(githubIdentifier),
			OAuthTokenID:      String(oauthTokenTest.ID),
			DisplayIdentifier: String(githubIdentifier),
			Branch:            String(githubBranch),
		},
		InitialVersion: String("1.0.0"),
		TestConfig: &RegistryModuleTestConfigOptions{
			TestsEnabled: Bool(true),
		},
	})

	if err != nil {
		oauthTokenTestCleanup()

		if orgCleanup != nil {
			orgCleanup()
		}

		t.Fatal(err)
	}

	return rm, func() {
		if err := client.RegistryModules.Delete(ctx, org.Name, rm.Name); err != nil {
			t.Errorf("Error destroying registry module! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Registry Module: %s\nError: %s", rm.Name, err)
		}

		oauthTokenTestCleanup()

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createRegistryModule(t *testing.T, client *Client, org *Organization, registryName RegistryName) (*RegistryModule, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()

	options := RegistryModuleCreateOptions{
		Name:         String(randomString(t)),
		Provider:     String("provider"),
		RegistryName: registryName,
	}

	if registryName == PublicRegistry {
		options.Namespace = "namespace"
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
	description := randomString(t)
	r, err := client.RunTasks.Create(ctx, org.Name, RunTaskCreateOptions{
		Name:        "tst-" + randomString(t),
		URL:         runTaskURL,
		Description: &description,
		Category:    "task",
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

func createRegistryProviderPlatform(t *testing.T, client *Client, provider *RegistryProvider, version *RegistryProviderVersion, targetOS, arch string) (*RegistryProviderPlatform, func()) {
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
		OS:       targetOS,
		Arch:     arch,
		Shasum:   genSha(t),
		Filename: randomString(t),
	}

	if targetOS == "" {
		options.OS = "linux"
	}

	if arch == "" {
		options.Arch = "amd64"
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

	state, err := os.ReadFile("test-fixtures/state-version/terraform.tfstate")
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

func createTeamProjectAccess(t *testing.T, client *Client, tm *Team, p *Project, org *Organization) (*TeamProjectAccess, func()) {
	var orgCleanup, tmCleanup, pCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	if tm == nil {
		tm, tmCleanup = createTeam(t, client, org)
	}

	if p == nil {
		p, pCleanup = createProject(t, client, org)
	}

	ctx := context.Background()
	tpa, err := client.TeamProjectAccess.Add(ctx, TeamProjectAccessAddOptions{
		Access:  *ProjectAccess(TeamProjectAccessAdmin),
		Team:    tm,
		Project: p,
	})
	if err != nil {
		t.Fatal(err)
	}

	return tpa, func() {
		if err := client.TeamProjectAccess.Remove(ctx, tpa.ID); err != nil {
			t.Errorf("Error removing team access! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"TeamAccess: %s\nError: %s", tpa.ID, err)
		}

		if tmCleanup != nil {
			tmCleanup()
		}

		if orgCleanup != nil {
			orgCleanup()
		}

		if pCleanup != nil {
			pCleanup()
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

func createTeamTokenWithOptions(t *testing.T, client *Client, tm *Team, options TeamTokenCreateOptions) (*TeamToken, func()) {
	var tmCleanup func()

	if tm == nil {
		tm, tmCleanup = createTeam(t, client, nil)
	}

	ctx := context.Background()
	tt, err := client.TeamTokens.CreateWithOptions(ctx, tm.ID, options)
	if err != nil {
		t.Fatal(err)
	}

	return tt, func() {
		if err := client.TeamTokens.DeleteByID(ctx, tt.ID); err != nil {
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
	options := VariableCreateOptions{
		Key:         String(randomString(t)),
		Value:       String(randomString(t)),
		Category:    Category(CategoryTerraform),
		Description: String(randomString(t)),
	}
	return createVariableWithOptions(t, client, w, options)
}

func createVariableWithOptions(t *testing.T, client *Client, w *Workspace, options VariableCreateOptions) (*Variable, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	if options.Key == nil {
		options.Key = String(randomString(t))
	}

	if options.Value == nil {
		options.Value = String(randomString(t))
	}

	if options.Description == nil {
		options.Description = String(randomString(t))
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
	v, err := client.Variables.Create(ctx, w.ID, options)
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
		if err := client.Workspaces.DeleteByID(ctx, w.ID); err != nil {
			t.Errorf("Error destroying workspace! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Workspace: %s\nError: %s", w.ID, err)
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
		options.VCSRepo = &VCSRepoOptions{}
	}

	options.VCSRepo.Identifier = String(githubIdentifier)
	options.VCSRepo.OAuthTokenID = String(oc.ID)

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

// This function is added to test setting up workspace's VCS connection via Github App Installation in place of
// Oauth token. For now the value of GHAInstallationID has to manually set to the correct value by the user.
func createWorkspaceWithGithubApp(t *testing.T, client *Client, org *Organization, options WorkspaceCreateOptions) (*Workspace, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	gHAInstallationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")

	if gHAInstallationID == "" {
		t.Skip("Export a valid GITHUB_APP_INSTALLATION_ID before running this test!")
	}

	options.VCSRepo.GHAInstallationID = String(gHAInstallationID)

	githubIdentifier := os.Getenv("GITHUB_POLICY_SET_IDENTIFIER")
	if githubIdentifier == "" {
		t.Fatal("Export a valid GITHUB_POLICY_SET_IDENTIFIER before running this test!")
	}

	if options.Name == nil {
		options.Name = String(randomString(t))
	}

	if options.VCSRepo == nil {
		options.VCSRepo = &VCSRepoOptions{}
	}

	options.VCSRepo.Identifier = String(githubIdentifier)

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

func applyVariableSetToWorkspace(t *testing.T, client *Client, vsID, wsID string) {
	if vsID == "" {
		t.Fatal("variable set ID must not be empty")
	}

	if wsID == "" {
		t.Fatal("workspace ID must not be empty")
	}

	opts := &VariableSetApplyToWorkspacesOptions{}
	opts.Workspaces = append(opts.Workspaces, &Workspace{ID: wsID})

	ctx := context.Background()
	if err := client.VariableSets.ApplyToWorkspaces(ctx, vsID, opts); err != nil {
		t.Fatalf("Error applying variable set %s to workspace %s: %v", vsID, wsID, err)
	}

	t.Cleanup(func() {
		removeOpts := &VariableSetRemoveFromWorkspacesOptions{}
		removeOpts.Workspaces = append(removeOpts.Workspaces, &Workspace{ID: wsID})
		if err := client.VariableSets.RemoveFromWorkspaces(ctx, vsID, removeOpts); err != nil {
			t.Errorf("Error removing variable set from workspace! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"VariableSet ID: %s\nError: %s", vsID, err)
		}
	})
}

func applyVariableSetToProject(t *testing.T, client *Client, vsID, prjID string) {
	t.Helper()
	if vsID == "" {
		t.Fatal("variable set ID must not be empty")
	}

	if prjID == "" {
		t.Fatal("project ID must not be empty")
	}

	opts := VariableSetApplyToProjectsOptions{}
	opts.Projects = append(opts.Projects, &Project{ID: prjID})

	ctx := context.Background()
	if err := client.VariableSets.ApplyToProjects(ctx, vsID, opts); err != nil {
		t.Fatalf("Error applying variable set %s to project %s: %v", vsID, prjID, err)
	}

	t.Cleanup(func() {
		removeOpts := VariableSetRemoveFromProjectsOptions{}
		removeOpts.Projects = append(removeOpts.Projects, &Project{ID: prjID})
		if err := client.VariableSets.RemoveFromProjects(ctx, vsID, removeOpts); err != nil {
			t.Errorf("Error removing variable set from project! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"VariableSet ID: %s\nError: %s", vsID, err)
		}
	})
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
// DEPRECATED : Please use the newSubscriptionUpdater instead.
func upgradeOrganizationSubscription(t *testing.T, _ *Client, organization *Organization) {
	newSubscriptionUpdater(organization).WithBusinessPlan().Update(t)
}

func createProject(t *testing.T, client *Client, org *Organization) (*Project, func()) {
	return createProjectWithOptions(t, client, org, ProjectCreateOptions{
		Name: randomStringWithoutSpecialChar(t),
	})
}

func createProjectWithOptions(t *testing.T, client *Client, org *Organization, options ProjectCreateOptions) (*Project, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	p, err := client.Projects.Create(ctx, org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	return p, func() {
		if err := client.Projects.Delete(ctx, p.ID); err != nil {
			t.Logf("Error destroying project! WARNING: Dangling resources "+
				"may exist! The full error is shown below.\n\n"+
				"Project ID: %s\nError: %s", p.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createTarGzipArchive(t *testing.T, files []string, outputPath string) {
	if len(files) == 0 {
		t.Fatal("files to archive are empty")
	}

	out, err := os.Create(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, filename := range files {
		func() {
			file, err := os.Open(filename)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			info, err := file.Stat()
			if err != nil {
				t.Fatal(err)
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				t.Fatal(err)
			}

			header.Name = filename
			err = tw.WriteHeader(header)
			if err != nil {
				t.Fatal(err)
			}

			_, err = io.Copy(tw, file)
			if err != nil {
				t.Fatal(err)
			}
		}()
	}

	t.Cleanup(func() {
		err := os.Remove(outputPath)
		if err != nil {
			t.Fatal("failed to delete archive: %w", err)
		}
	})
}

func waitForSVOutputs(t *testing.T, client *Client, svID string) {
	t.Helper()

	_, err := retryPatiently(func() (interface{}, error) {
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
}

func waitForRunLock(t *testing.T, client *Client, workspaceID string) {
	t.Helper()
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
}

func retryTimes(maxRetries, secondsBetween int, f retryableFn) (interface{}, error) {
	tick := time.NewTicker(time.Duration(secondsBetween) * time.Second)
	retries := 0

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

func retryPatiently(f retryableFn) (interface{}, error) { //nolint
	return retryTimes(39, 3, f) // 40 attempts over 120 seconds
}

func retry(f retryableFn) (interface{}, error) { //nolint
	return retryTimes(9, 3, f) // 10 attempts over 30 seconds
}

func genSha(t *testing.T) string {
	t.Helper()
	h := hmac.New(sha256.New, []byte("secret"))
	_, err := h.Write([]byte("data"))
	if err != nil {
		t.Fatalf("error writing hmac: %s", err)
	}
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

// genSafeRandomTerraformVersion returns a random version number of the form
// `1.0.<RANDOM>`, which HCP Terraform won't ever select as the latest available
// Terraform. (At the time of writing, a fresh HCP Terraform instance will include
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

// createAdminSentinelVersion returns a random version number of the form
// `0.0.<RANDOM>`
func createAdminSentinelVersion() string {
	rInt := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	return fmt.Sprintf("0.0.%d", rInt)
}

// createAdminOPAVersion returns a random OPA version number of the form
// `0.0.<RANDOM>`
func createAdminOPAVersion() string {
	rInt := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	return fmt.Sprintf("0.0.%d", rInt)
}

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func randomStringWithoutSpecialChar(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	uuidWithoutHyphens := strings.ReplaceAll(v, "-", "")
	return uuidWithoutHyphens
}

func randomKeyValue(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	uuidWithoutHyphens := strings.ReplaceAll(v, "-", "")
	return "t" + uuidWithoutHyphens
}

func containsProject(pl []*Project, str string) bool {
	for _, p := range pl {
		if p.Name == str {
			return true
		}
	}
	return false
}

func randomSemver(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%d.%d.%d", rand.Intn(99)+3, rand.Intn(99)+1, rand.Intn(99)+1)
}

// skips a test if the environment is for HCP Terraform.
func skipUnlessEnterprise(t *testing.T) {
	t.Helper()
	if !enterpriseEnabled() {
		t.Skip("Skipping test related to HCP Terraform. Set ENABLE_TFE=1 to run.")
	}
}

// skips a test if the environment is for Terraform Enterprise
func skipIfEnterprise(t *testing.T) {
	t.Helper()
	if enterpriseEnabled() {
		t.Skip("Skipping test related to Terraform Enterprise. Set ENABLE_TFE=0 to run.")
	}
}

// skips a test if the underlying beta feature is not available.
// **Note: ENABLE_BETA is always disabled in CI, so ensure you:
//
//  1. Run tests locally and paste the test output in the resulting pull request
//  2. Remove the beta requirements of your feature from go-tfe once the feature is generally available.
//
// See CONTRIBUTING.md for details
func skipUnlessBeta(t *testing.T) {
	t.Helper()
	if !betaFeaturesEnabled() {
		t.Skip("Skipping test related to a HCP Terraform beta feature. Set ENABLE_BETA=1 to run.")
	}
}

// skips a test if the architecture is not linux_amd64
func skipUnlessLinuxAMD64(t *testing.T) {
	t.Helper()
	if !linuxAmd64() {
		t.Skip("Skipping test if architecture is not linux_amd64")
	}
}

// Temporarily skip a test that may be experiencing API errors. This method
// purposefully errors after the set date to remind contributors to remove this check
// and verify that the API errors are no longer occurring.
func skipUnlessAfterDate(t *testing.T, d time.Time) {
	today := time.Now()
	if today.After(d) {
		t.Fatalf("This test was temporarily skipped and has now expired. Remove this check to run this test.")
	} else {
		t.Skipf("Temporarily skipping test due to external issues: %s", t.Name())
	}
}

func linuxAmd64() bool {
	return runtime.GOOS == "linux" && runtime.GOARCH == "amd64"
}

// Checks to see if ENABLE_TFE is set to 1, thereby enabling enterprise tests.
func enterpriseEnabled() bool {
	return os.Getenv("ENABLE_TFE") == "1"
}

// Checks to see if ENABLE_BETA is set to 1, thereby enabling tests for beta features.
func betaFeaturesEnabled() bool {
	return os.Getenv("ENABLE_BETA") == "1"
}

// isEmpty gets whether the specified object is considered empty or not.
func isEmpty(object interface{}) bool {
	// get nil case out of the way
	if object == nil {
		return true
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	// collection types are empty when they have no element
	case reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0
	// pointers are empty if nil or if the value they point to is empty
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return isEmpty(deref)
	// for all other types, compare against the zero value
	// array types are empty when they match their zero-initialized state
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(object, zero.Interface())
	}
}

// requireExactlyOneNotEmpty accepts any number of values and calls t.Fatal if
// less or more than one is empty.
func requireExactlyOneNotEmpty(t *testing.T, v ...any) {
	if len(v) == 0 {
		t.Fatal("Expected some values for requireExactlyOneNotEmpty, but received none")
	}

	empty := 0
	for _, value := range v {
		if isEmpty(value) {
			empty += 1
		}
	}

	if empty != len(v)-1 {
		t.Fatalf("Expected exactly one value to not be empty, but found %d empty values", empty)
	}
}

func runTaskCallbackMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			return
		}
		if r.Header.Get("Accept") != ContentTypeJSONAPI {
			t.Fatalf("unexpected accept header: %q", r.Header.Get("Accept"))
		}
		if r.Header.Get("Authorization") != fmt.Sprintf("Bearer %s", testTaskResultCallbackToken) {
			t.Fatalf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Authorization") == fmt.Sprintf("Bearer %s", testInitialClientToken) {
			t.Fatalf("authorization header is still the initial one: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") != "go-tfe" {
			t.Fatalf("unexpected user agent header: %q", r.Header.Get("User-Agent"))
		}
	}))
}

// Useless key but enough to pass validation in the API
const testGpgArmor string = `
-----BEGIN PGP PUBLIC KEY BLOCK-----

mQINBGKnWEYBEACsTJ9HEUrXBaBvQvXZAXEIMWloG96MVAdCj547jJviSS4TqMIQ
EST2pzDq7lEpqL+JkW3ptyLEAeQs6gJJeuhODGm2EcxjJ9/JM4ZH+p9zq2wBeXVe
0XJcP3HD8/7MesjMyGSsoX7tR7TcIhs5Y7zS+/L1xnoReYUsBgC6QdqjQwkuntaq
2y6yxdYG4gVlxb4yA0Ga6Qfy0VGIKjbCdPqCRyJ76YHE3t+Skq9oDCOV3VSiwKsU
V/ivf/MVZ1GyE03anW0+poVK38Ekogsd2+34uEjusbuoJGmHzh/20IDS8VnxQHIY
qdVwcZrW+a3O6nexL4dJJGMfXMbCdS87FxpSnC1FDGMSJ2c5cxlMuKuDboTpbRy5
Dd80p6voJQcLcpr0hKYIwwDGJYE336KMFqf/apCc6HbCFfN8kCYg3K7+4yganRWu
h/9qIhP0QaYOYEQl4RdjJTSyJSP3srAJ3F5OmrAhRXlHlLo1p00zxFxG7ZcJER6l
+uRubtL9WN2kgGbr9NDJbz/HeOTjJhCASdQuzstcL8RrFMDftE/P2K8LnkxUNIbT
dhZtwvkhnyIwOZIHwsQddeJboeHD445SlHJ+4vFsPKRTuNu5u9GhVSyZhoHmdeH0
FheD8p43+BKZ7KmD4xd+zfCQE1xO2cO9ZrCNV2hs9UVFbgZfjokqWkuHJQARAQAB
tBNmb28gPGFkbWluQHRmZS5jb20+iQJRBBMBCAA7FiEE/2esSrAATXzEQSanE9/s
yjtYzkoFAmKnWEYCGwMFCwkIBwICIgIGFQoJCAsCBBYCAwECHgcCF4AACgkQE9/s
yjtYzkq01g/9EgnW0NBD4DdtQSHg5jya0lx5iNHLK+umwL2x7abcSQ9iTIylhbHP
+he6jS/p4yzK7Gf7S+W3D9EZ58KrTMhu85iLr0uZ947pEbC0kDlQGkIfiK0CAyq2
IDj1RFgmeM0E2LkPOYCM+JPeBC9nZduFMYY9eFhCZXJ3ua1DP37ZBdZbjuImbiQ5
abt75a89NbQI3KRaACzqEjFpRYuoxbh8RznkTFf57AFzt4yMWy+4l47GSXTE8boS
1P7ZOfvJPuh2RRN9sSe0eTPCYnnSxPPo0LvgqSnLSk9yc65nkPZmlSXVdswV5Le+
7LlKG+rTwXljfGwLmj0VNn2gGCKe5IHs8FKt3parSiQOu4MXHCHshSQDEvXyIugJ
i2V2pcw4Hi6f2Znh3YYJamL6fDwCpDcTOCxZbvFi4OuBzbWcDLP1k52k3ZyYce92
1CK84HWtoRseNlVt1rieClPZH5T4b0HMPBWKK39/r+RABJDAfdGtn2ulKXK2JugH
AYXlhY9xh9+r1O7tsqExGkEYnp7nI0ArauJhIUWZybpGpPYP99kK4F64E4DRu1si
/3eeYoqKY1jAHoebRzn3XcRg5kro/lJYQQIhT4fHt5sAc/e8gDdaQaDPIftsmu7K
w4e6pMyztiMfRw7w0ZSjGlPsl0NiXA3nuG966gx4Bnx/ddJIHrghAi25Ag0EYqdY
RgEQAOGONFP+z45+9gvnT1yd9sJLqxYhtj5QRxKkXkLARPd0Yjdyff/lVd1YPtZ7
slLuEGlBDKdB6aIeu3b1C95Ie3qbTIwIp6ZYKGqUEwGW/0sPtBqqXanVrQkrY4ho
lqejgPraFgF6sDGrSxG7b8W985NJwKcm8Lx1/x4ZwvpUrQlCL4UajJcECmjVqU/e
ofjWZFZl7eR2oYh2BBzvA8mwkVKXs6kTGWLkK7VDeR2lCRl2fk4+5DydbOMIZXxT
jmYR8iu2Mr+gt//VmvvBjlFMI05kwD9iG3SRYBwpYEXETKCE12KKqcbhP/bwahIB
bcsaQkoky9jgtp7tizduPOkjkGhT9kF8L1O0VGxek40L7+QIDEnVHMAH5hSLmgau
vJF+Bd0W/TRZbmAJXoWPreftVTmWH7xH4N7v+3dvWziIJPt+N/1HHeZXBojJJAVk
6C+t1KpsSwGzzOjdsQVCklT7D4PmWtzz6FAjImPSbk5LbiVWis/lH+SEVZS4sG7j
pR3vRjUZTjCi/8CmHTjiWXL7g9kkt//a5Av3iArQq0pv0QNPG/uPeN2QTnkz5DAo
kM/qUx/G59i8AfEH2myh9oPCOzb3yFOsK9G/2Sy05cfdLozddHwt+hJVPx1Od9Nr
HAJMQspr9AaZPB9FnAa0Bv/RNEGJv6LJwzVWJkezL2wQAZdlABEBAAGJAjYEGAEI
ACAWIQT/Z6xKsABNfMRBJqcT3+zKO1jOSgUCYqdYRgIbDAAKCRAT3+zKO1jOSq9E
D/4hlNaCwY/etk7ZvMe4pupQATzrZF58d2qjx4niMd3CvCWmbrWMmoNxBjECXc8H
kp+0NURFFc/wiCn/Q6dhrMxKVCpsWpHA1Doi/vtzQtM081Ib6uIX6L6liyUexW1l
tvJwPurqJJVBW3ikOjICCnv70tp2zaS47uQjyFGTnzglIU961EXCWdNjH1vm8bFJ
BxXN87gHXhUUw8GZ3d2V75TAJIEqRVV+eI4flXcJ4Ld+Zbt2EiMwtQ05XCc8bgsc
QzZFizw936bC5Py7Iu6aEaShFlZlz8LgYcId32UYh5PG1xGNZv0C9Z/PJECx5zcx
RJszDpm3erpmdkkJf9UBuhjjTdQ9gheFjZRDi/rVJ0JPVxD7HTzEAWd5MqFXqh0V
j2xG1FhtfxSaMf9rsJjtwewLPyZylSuz2erz1j80Hx3Q6eSIDsNjnDTtfh9Z8gXz
gvF7mSC0lZu/RvDSRyHfCw4zCQ04HieIvq3hZLy+QS11ykJTSKAePKk77EmwtoLd
Je9n9FCKhLknUp1/dsu0lsznvttOLwYy6xFP4JNPgiq6iYlVHs417oib67DrGlsI
3Ki44OESW/vL3WAC091TOF4OYgGw+TMauB8SxZo0PLXrIwKeBsQEB4tf6bX66OvJ
UFpas2r53xTaraRDpu6+u66hLY+/XV9Uf5YzETuPQnX/nw==
=bBSS
-----END PGP PUBLIC KEY BLOCK-----
`
