// Copyright IBM Corp. 2018, 2026

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/account"
	"github.com/microsoft/kiota-abstractions-go/serialization"
)

func main() {
	oldPassword := flag.String("old-password", "", "Current account password")
	newPassword := flag.String("new-password", "", "New account password")
	flag.Parse()

	if *oldPassword == "" || *newPassword == "" {
		flag.Usage()
		os.Exit(2)
	}

	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("TFE_TOKEN"),
		Address: os.Getenv("TFE_ADDRESS"),
	})
	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
	}

	attributes := account.NewPasswordPatchRequestBody_data_attributes()
	attributes.SetCurrentPassword(oldPassword)
	attributes.SetPassword(newPassword)
	attributes.SetPasswordConfirmation(newPassword)

	data := account.NewPasswordPatchRequestBody_data()
	data.SetAttributes(attributes)

	body := account.NewPasswordPatchRequestBody()
	body.SetData(data)

	response, err := client.API.Account().Password().Patch(context.Background(), body, nil)
	if err != nil {
		log.Fatalf("Error changing account password: %s", err)
	}

	buffer, err := serialization.SerializeToJson(response)
	if err != nil {
		log.Fatalf("Error serializing response: %s", err)
	}

	fmt.Println(string(buffer))
}
