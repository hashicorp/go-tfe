package main

import (
	"context"
	"log"
	"os"
	"time"

	tfe "github.com/hashicorp/go-tfe"
)

func main() {
	config := &tfe.Config{
		Address:    os.Getenv("TFE_ADDRESS"),
		Token:      os.Getenv("TFE_TOKEN"),
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context
	ctx := context.Background()

	// Create a new policy set
	ps, err := client.PolicySets.Create(ctx, os.Getenv("TFE_ORGANIZATION"), tfe.PolicySetCreateOptions{
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
		log.Print("The policy set version Type is: ", psv.Data.Type, "\n")
		log.Print("The policy set version ID is: ", psv.Data.ID, "\n")
		log.Print("The linked policy set ID is: ", psv.Data.Relationships.PolicySet.Data.ID, "\n")
		log.Print("The upload link is:", psv.Data.Links.Upload, "\n")
		log.Print("The upload status is: ", psv.Data.Attributes.Status)
	}

	// Get the upload Link
	uploadLink := psv.Data.Links.Upload

	// Log upload URL
	log.Print("The upload URL is:", uploadLink, "\n")

	// Upload a policy set to the upload link
	err = client.PolicySetVersions.Upload(ctx, uploadLink, "../../test-fixtures/policy-set-version")
	if err != nil {
		log.Fatal(err)
	}

	// Try to read the policy set version
	for i := 0; ; i++ {
		psv, err = client.PolicySetVersions.Read(ctx, psv.Data.ID)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Print("The upload status is: ", psv.Data.Attributes.Status)
		}

		if psv.Data.Attributes.Status == tfe.PolicySetVersionReady {
			break
		}

		if i > 10 {
			log.Fatal("Timeout waiting for the policy set version to be uploaded")
		}

		time.Sleep(1 * time.Second)
	}

	// Delete the policy set
	// Note that there is no API to delete policy set versions
	err = client.PolicySets.Delete(ctx, ps.ID)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Successfully deleted policy set", ps.ID)
	}
}
