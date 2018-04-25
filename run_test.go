package tfe

import (
	"testing"
)

func TestCreateRun(t *testing.T) {
	client := testClient(t)

	ws, wsCleanup := createWorkspace(t, client, nil)
	defer wsCleanup()

	input := &CreateRunInput{
		WorkspaceID: ws.ID,
		//ConfigurationVersionID: String(""),
		Message: String("yo"),
		Comment: String("sup"),
	}

	run, err := client.CreateRun(input)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", run)
}
