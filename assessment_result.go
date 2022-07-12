package tfe

// Compile-time proof of interface implementation.
var _ AsessmentResults = (*assessmentResults)(nil)

type AssessmentResults interface {
	// Read a plan by its ID.
	Read(ctx context.Context, assessmentResultID string) (*AssessmentResult, error)

	// Find a current AssssmentResult for a Wokrspace
	ReadCurrentForWorkspace(ctx context.Context, workspaceID string) (*AssessmentResult, error)

	// Retrieve the JSON execution plan
	//ReadJSONOutput(ctx context.Context, assessmentResultID string) ([]byte, error)
}

type assessmentResults struct {
	client *Client
}

//
// No list support yet
//

type AssessmentResult struct {
	ID        string    `jsonapi:"primary,plans"`
	CreatedAt time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	Drifted   bool      `jsonapi:"attr,drifted"`
	Succeeded bool      `jsonapi:"attr,succeeded"`

	// Relations
	Workspace *Workspace `jsonapi:"relation,workspace"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

func (s *assessmetResults) Read(ctx context.Context, assessmentResultID string) (*AssessmentResult, error) {
	// not implemented
}
func (s *assessmetResults) ReadCurrentForWorkspace(ctx context.Context, workspaceID string) (*AssessmentResult, error) {
	// not implemented
}
