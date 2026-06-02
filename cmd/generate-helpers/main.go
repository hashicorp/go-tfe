// generate-helpers reads openapi/spec.json and emits helpers/zz_generated_helpers.go.
// generate-helpers only generates helpers for schemas that are found in components/schemas and
// have a JSON:API request-body shape with at least one eligible field. Schemas that don't match
// this shape are not supported; move them to the supported shape in the upstream spec first.

// Run via: go run ./cmd/generate-helpers  (from the module root)
package main

import (
	"encoding/json"
	"log"
	"os"
)

func main() {
	data, err := os.ReadFile("openapi/spec.json")
	if err != nil {
		log.Fatalf("reading openapi/spec.json: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		log.Fatalf("parsing spec: %v", err)
	}

	schemas, err := parseSchemas(spec)
	if err != nil {
		log.Fatalf("parsing schemas: %v", err)
	}

	if err := os.MkdirAll("helpers", 0o755); err != nil {
		log.Fatalf("creating helpers dir: %v", err)
	}

	outPath := "helpers/zz_generated_helpers.go"
	if err := generateHelpers(schemas, outPath); err != nil {
		log.Fatalf("generating helpers: %v", err)
	}

	log.Printf("wrote %s (%d schemas)", outPath, len(schemas))
}
