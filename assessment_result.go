package tfe

// Compile-time proof of interface implementation.
var _ AsessmentResults = (*assessmentResults)(nil)

type AssessmentResults interface {
	// Read an AssessmentResult by its ID.
	Read(ctx context.Context, assessmentResultID string) (*AssessmentResult, error)

	// Find a current AssssmentResult for a Wokrspace
	ReadCurrentForWorkspace(ctx context.Context, workspaceID string) (*AssessmentResult, error)

	// Retrieve the assessment log output
	Logs(ctx context.Context, assessmentResultID string) (io.Reader, error)

	// Retrieve the JSON execution plan
	ReadJSONOutput(ctx context.Context, assessmentResultID string) ([]byte, error)

	// Retrieve the JSON schema
	ReadJSONSchema(ctx context.Context, assessmentResultID string) ([]byte, error)
}

type assessmentResults struct {
	client *Client
}

//
// No list support yet
//

// Used in conjunction with AssessmentResult.Links as lookup keys
const (
	AssessmentJSONOutputLinkKey = "json-output"
	AssessmentJSONSchemaLinkKey = "json-schema"
	AssessmentLogLinkKey        = "log-output"
)

type AssessmentResult struct {
	ID        string    `jsonapi:"primary,assessment-results"`
	CreatedAt time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	Drifted   bool      `jsonapi:"attr,drifted"`
	Succeeded bool      `jsonapi:"attr,succeeded"`

	// Relations
	Workspace *Workspace `jsonapi:"relation,workspace"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

// Read an assessment result by its ID.
func (s *assessmetResults) Read(ctx context.Context, assessmentResultID string) (*AssessmentResult, error) {
	if !validStringID(&assessmentResultID) {
		return nil, ErrInvalidAssessmentResultID
	}

	u := fmt.Sprintf("assessment-results/%s", url.QueryEscape(assessmentResultID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	a := &AssessmentResult{}
	err = req.Do(ctx, a)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Read the current relevant assessment result for a workspace.
func (s *assessmetResults) ReadCurrentForWorkspace(ctx context.Context, workspaceID string) (*AssessmentResult, error) {
	if !validStringID(&workspaceID) {
		return nil, ErrInvalidWorkspaceID
	}

	u := fmt.Sprintf("workspaces/%s/current-assessment-results", url.QueryEscape(workspaceID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	a := &AssessmentResult{}
	err = req.Do(ctx, a)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Retrieve the JSON execution plan
func (s *assessmetResults) ReadJSONOutput(ctx context.Context, assessmentResultID string) ([]byte, error) {
	if !validStringID(&assessmentResultID) {
		return nil, ErrInvalidAssessmentResultID
	}

	u := fmt.Sprintf("assessment-results/%s/json-output", url.QueryEscape(assessmentResultID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = req.Do(ctx, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Retrieve the JSON schema from the assessment
func (s *assessmetResults) ReadJSONSchema(ctx context.Context, assessmentResultID string) ([]byte, error) {
	if !validStringID(&assessmentResultID) {
		return nil, ErrInvalidAssessmentResultID
	}

	u := fmt.Sprintf("assessment-results/%s/json-schema", url.QueryEscape(assessmentResultID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = req.Do(ctx, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Logs retrieves the logs of a assessment.
func (s *assessmentResults) Logs(ctx context.Context, assessmentResultID string) (io.Reader, error) {
	if !validStringID(&assessmentResultID) {
		return nil, ErrInvalidAssessmentResultID
	}

	// Get the assessment result to make sure it exists.
	a, err := s.Read(ctx, assessmentResultID)
	if err != nil {
		return nil, err
	}

	// Return an error if the log URL is empty.
	if a.Links[AssessmentLogLinkKey] == "" {
		return nil, fmt.Errorf("assessment result %s does not have a log URL", assessmentResultID)
	}

	u, err := url.Parse(a.Links[AssessmentLogLinkKey])
	if err != nil {
		return nil, fmt.Errorf("invalid log URL: %w", err)
	}

	done := func() (bool, error) {
		p, err := s.Read(ctx, a.ID)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return &LogReader{
		client: s.client,
		ctx:    ctx,
		done:   done,
		logURL: u,
	}, nil
}
