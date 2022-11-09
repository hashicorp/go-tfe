package tfe

import (
	"context"
	"fmt"
	"testing"
)

func TestList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	po := policyOutcome{
		client: client,
	}
	opts := &PolicyOutcomeListOptions{
		Filter: map[string]PolicyOutcomeListFilter{
			"0": {Status: "errored"},
			"1": {Status: "failed", EnforcementLevel: "mandatory"},
		},
		//Filter: []PolicyOutcomeListFilter{
		//	{Status: "errored"},
		//	{Status: "failed", EnforcementLevel: "mandatory"},
		//},
	}
	r, err := po.List(ctx, "pol1244", opts)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(r)
}
