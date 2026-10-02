// Copyright IBM Corp. 2018, 2026

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/account"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	khttp "github.com/microsoft/kiota-http-go"
)

func main() {
	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("TFE_TOKEN"),
		Address: os.Getenv("TFE_ADDRESS"),
	})
	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
	}

	inspection := khttp.NewHeadersInspectionOptions()
	inspection.InspectResponseHeaders = true
	configuration := abstractions.RequestConfiguration[account.DetailsRequestBuilderGetQueryParameters]{
		Options: []abstractions.RequestOption{inspection},
	}

	_, err = client.API.Account().Details().Get(context.Background(), &configuration)
	if err != nil {
		log.Fatalf("Error getting account details: %s", err)
	}

	headers := inspection.GetResponseHeaders()
	for _, name := range headers.ListKeys() {
		fmt.Printf("%s: %v\n", name, headers.Get(name))
	}
}
