// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	tfe "github.com/optable/go-tfe"
)

var (
	pollInterval = 500 * time.Millisecond
)

// Diagnostic represents a diagnostic type message from Terraform, which is how errors
// are usually represented.
type Diagnostic struct {
	Severity string           `json:"severity"`
	Summary  string           `json:"summary"`
	Detail   string           `json:"detail"`
	Address  string           `json:"address,omitempty"`
	Range    *DiagnosticRange `json:"range,omitempty"`
}

// Pos represents a position in the source code.
type Pos struct {
	// Line is a one-based count for the line in the indicated file.
	Line int `json:"line"`

	// Column is a one-based count of Unicode characters from the start of the line.
	Column int `json:"column"`

	// Byte is a zero-based offset into the indicated file.
	Byte int `json:"byte"`
}

// DiagnosticRange represents the filename and position of the diagnostic subject.
type DiagnosticRange struct {
	Filename string `json:"filename"`
	Start    Pos    `json:"start"`
	End      Pos    `json:"end"`
}

// For full decoding, see https://github.com/optable/terraform/blob/main/internal/command/jsonformat/renderer.go
type JSONLog struct {
	Message    string      `json:"@message"`
	Level      string      `json:"@level"`
	Timestamp  string      `json:"@timestamp"`
	Type       string      `json:"type"`
	Diagnostic *Diagnostic `json:"diagnostic"`
}

// Given a
func logErrorsOnly(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		var jsonLog JSONLog
		err := json.Unmarshal([]byte(scanner.Text()), &jsonLog)
		// It's possible this log is not encoded as JSON at all, so errors will be ignored.
		if err == nil && jsonLog.Level == "error" {
			fmt.Println()
			fmt.Println("--- Error Message")
			fmt.Println(jsonLog.Message)
			fmt.Println("---")
			fmt.Println()
			if jsonLog.Type == "diagnostic" {
				fmt.Println("--- Diagnostic Details")
				fmt.Println(jsonLog.Diagnostic.Detail)
				fmt.Println("---")
				fmt.Println()
			}
		}
	}
}

func logRunErrors(ctx context.Context, client *tfe.Client, run *tfe.Run) {
	var reader io.Reader
	var err error

	if run.Apply != nil && run.Apply.Status == tfe.ApplyErrored {
		log.Printf("Reading apply logs from %q", run.Apply.LogReadURL)
		reader, err = client.Applies.Logs(ctx, run.Apply.ID)
	} else if run.Plan != nil && run.Plan.Status == tfe.PlanErrored {
		log.Printf("Reading apply logs from %q", run.Plan.LogReadURL)
		reader, err = client.Plans.Logs(ctx, run.Plan.ID)
	} else {
		log.Fatal("Failed to find an errored plan or apply.")
	}

	if err != nil {
		log.Fatal("Failed to read error log: ", err)
	}

	logErrorsOnly(reader)
}

func readRun(ctx context.Context, client *tfe.Client, id string) *tfe.Run {
	r, err := client.Runs.ReadWithOptions(ctx, id, &tfe.RunReadOptions{
		Include: []tfe.RunIncludeOpt{tfe.RunApply, tfe.RunPlan},
	})
	if err != nil {
		log.Fatal("Failed to read specified run: ", err)
	}
	return r
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Printf("\t%s <run ID>\n", os.Args[0])
		os.Exit(1)
	}

	ctx := context.Background()
	client, err := tfe.NewClient(&tfe.Config{
		Address:           "https://app.terraform.io",
		RetryServerErrors: true,
	})
	if err != nil {
		log.Fatal("Failed to initialize client: ", err)
	}

	r := readRun(ctx, client, os.Args[1])

poll:
	for {
		<-time.After(pollInterval)

		r := readRun(ctx, client, r.ID)

		switch r.Status {
		case tfe.RunApplied:
			fmt.Println("Run finished!")
		case tfe.RunErrored:
			fmt.Println("Run had errors!")
			logRunErrors(ctx, client, r)
			break poll
		default:
			fmt.Printf("Waiting for run to error... Run status was %q...\n", r.Status)
		}
	}
}
