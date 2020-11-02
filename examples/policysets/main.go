package main

import (
	"context"
	"log"
	"time"

	tfe "github.com/hashicorp/go-tfe"
)

func main() {
	config := &tfe.Config{
		Token: "insert-your-token-here",
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context
	ctx := context.Background()

	// Create a new policy set
	ps, err := client.PolicySets.Create(ctx, "org-name", tfe.PolicySetCreateOptions{
		Name: tfe.String("my-policy-set-tst"),
	})
	if err != nil {
		log.Fatal(err)
	} else {
		log.Print("The policy set ID is:", ps.ID, "\n")
	}

	// Create a policy set version
	psv, err := client.PolicySetVersions.Create(ctx, ps.ID, tfe.PolicySetVersionCreateOptions{})
	if err != nil {
		log.Println("Failed creating policy set version")
		log.Fatal(err)
	} else {
		log.Print("The policy set version ID is:", psv.ID, "\n")
		log.Print("The linked policy set ID is:", psv.PolicySet.ID, "\n")
	}

	// Upload a policy set version tar ball
	uploadLink := ""
	for k, v := range *psv.Links {
		if k == "upload" {
			uploadLink = v.(string)
		}
	}

	// Log upload URL
	log.Print("The upload URL is:", uploadLink, "\n")

	// Upload a policy set to the upload link
	err = client.PolicySetVersions.Upload(ctx, uploadLink, "../../test-fixtures/policy-set-version")
	if err != nil {
		log.Fatal(err)
	}

	// Try to read the policy set version
	for i := 0; ; i++ {
		psv, err = client.PolicySetVersions.Read(ctx, psv.ID)
		if err != nil {
			log.Fatal(err)
		}

		if psv.Status == tfe.PolicySetVersionUploaded {
			break
		}

		if i > 10 {
			log.Fatal("Timeout waiting for the policy set version to be uploaded")
		}

		time.Sleep(1 * time.Second)
	}

}
