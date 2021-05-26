package tfe

// Compile-time proof of interface implementation.
var _ PolicySetVersions = (*policySetVersions)(nil)

// PolicySetVersions describes all the Policy Set Version related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/policy-sets.html#create-a-policy-set-version

type PolicySetVersions interface {
	// Create is used to create a new Policy Set Version.
	Create(ctx context.Context, policySetID string) (*PolicySetVersion, error)

	// Read is used to read a Policy Set Version by the policy set ID.
	Read(ctx context.Context, policySetID string) (*PolicySetVersion, error)

	// Upload a tarball to the Policy Set Version.
	Upload(ctx context.Context, policySetID string, context []byte) (*PolicySetVersion, error)
}

// policySetVersions implements Policy Set Versions.
type policySetParameters struct {
	client *Client
}

// PolciySetVersionSource represents a source type of a policy set version.
type PolciySetVersionSource string

// List all available run sources.
const (
	PolciySetVersionSourceAPI PolciySetVersionSource = "tfe-api"
	PolciySetVersionSourceUI  PolciySetVersionSource = "tfe-ui"
)

type PolicySetVersion struct {
	ID     string `jsonapi:"primary,policy-set-versions"`
	Source string `jsonapi:"attr,source"`
	Status string `jsonapi:"attr,status"`
	// TODO: StatusTimestamps  string `jsonapi:"attr,status-timestamps"`
	Error     string    `jsonapi:"attr,error"`
	CreatedAt time.Time `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt time.Time `jsonapi:"attr,updated-at,iso8601"`

	// Relations
	PolicySet *PolicySet `jsonapi:"relation,policy-set"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

func (p *policySetVersions) Create(ctx context.Context, policySetID string) (*PolicySetVersion, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}

	u := fmt.Sprintf("policy-sets/%s/versions", url.QueryEscape(policySetID))
	req, err := p.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	psv := &PolicySetVersion{}
	err = p.client.do(ctx, req, psv)
	if err != nil {
		return nil, err
	}

	return psv, nil
}

func (p *policySetVersions) Read(ctx context.Context, policySetID string) (*PolicySetVersion, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}

	u := fmt.Sprintf("policy-set-versions/%s", url.QueryEscape(policySetID))
	req, err := p.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	psv := &PolicySetVersion{}
	err = p.client.do(ctx, req, psv)
	if err != nil {
		return nil, err
	}

	return psv, nil
}

func (p *policySetVersions) Upload(ctx context.Context, policySetID string, content []byte) (*PolicySetVersion, error) {
	if !validStringID(&policyID) {
		return errors.New("invalid value for policy ID")
	}

	psv, err := p.Read(ctx, policySetID)
	if err != nil {
		return err
	}
	uploadURL, ok := psv.Links["upload"].(string)
	if !ok {
		return fmt.Errorf("The Policy Set Version does not contain an upload link.")
	}

	req, err := p.client.newRequest("PUT", uploadURL, content)
	if err != nil {
		return err
	}

	return p.client.do(ctx, req, nil)
}
