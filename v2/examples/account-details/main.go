// Copyright IBM Corp. 2018, 2026

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-tfe/v2"
	"github.com/microsoft/kiota-abstractions-go/serialization"
)

func main() {
	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("TFE_TOKEN"),
		Address: os.Getenv("TFE_ADDRESS"),
	})
	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
	}

	response, err := client.API.Account().Details().Get(context.Background(), nil)
	if err != nil {
		log.Fatalf("Error getting account details: %s", err)
	}

	buffer, err := serialization.SerializeToJson(response)
	if err != nil {
		log.Fatalf("Error serializing response: %s", err)
	}

	fmt.Println(string(buffer))
}
