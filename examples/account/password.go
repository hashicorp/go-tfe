package account

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/api/models"
	"github.com/microsoft/kiota-abstractions-go/serialization"
)

type accountPasswordCommand struct{}

var _ cli.Command = accountPasswordCommand{}

func AccountChangePasswordCommandFactory() (cli.Command, error) {
	return &accountPasswordCommand{}, nil
}

func (accountPasswordCommand) Help() string {
	return "Change your account password"
}

func (accountPasswordCommand) Synopsis() string {
	return "Change your account password"
}

func (accountPasswordCommand) changePasswordModelHelper(oldPassword, newPassword *string) *models.Account_password {
	pw := models.NewAccount_password()
	pwd := models.NewAccount_password_data()

	pwda := models.NewAccount_password_data_attributes()
	pwda.SetCurrentPassword(oldPassword)
	pwda.SetPassword(newPassword)
	pwda.SetPasswordConfirmation(newPassword)

	t := models.USERS_ACCOUNT_PASSWORD_DATA_TYPE
	pwd.SetTypeEscaped(&t)
	pwd.SetAttributes(pwda)
	pw.SetData(pwd)

	return pw
}

func (c accountPasswordCommand) Run(args []string) int {
	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("TFE_TOKEN"),
		Address: os.Getenv("TFE_ADDRESS"),
	})

	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
		return 1
	}

	passwordFlags := flag.NewFlagSet("password", flag.ContinueOnError)
	oldPassword := passwordFlags.String("old-password", "", "Old password")
	newPassword := passwordFlags.String("new-password", "", "New password")

	if err := passwordFlags.Parse(args); err != nil {
		log.Fatalf("Error parsing flags: %s", err)
		return 1
	}

	if oldPassword == nil || *oldPassword == "" || newPassword == nil || *newPassword == "" {
		fmt.Println("Both --old-password and --new-password are required")
		passwordFlags.Usage()
		return 1
	}

	ctx := context.Background()

	response, err := client.API.Account().Password().Patch(ctx, c.changePasswordModelHelper(oldPassword, newPassword), nil)
	if err != nil {
		var apiErr *tfe.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("API returned an error status: %d, details:\n- %s", apiErr.StatusCode, strings.Join(apiErr.Details, "\n- "))
		}
		log.Fatalf("API returned an error status: %s", err)
		return 1
	}

	// Serialize the response to JSON for display
	buffer, err := serialization.SerializeToJson(response)
	if err != nil {
		log.Fatalf("Error serializing response: %s", err)
		return 1
	}

	fmt.Println(string(buffer))
	return 0
}
