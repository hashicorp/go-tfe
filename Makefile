.PHONY: vet fmt lint test spec api

vet:
	go vet

fmt:
	gofmt -s -l -w .

fmtcheck:
	./scripts/gofmtcheck.sh

lint:
	cd v2 && golangci-lint run

lint_v1:
	golangci-lint run

test:
	cd v2 && go test ./... $(TESTARGS)

spec:
	mkdir -p v2/openapi
	mkdir -p v2/api

	curl -o ./v2/openapi/spec.json https://app.terraform.io/openapi/prerelease.json

api: spec
	docker run --rm \
		--user "$$(id -u):$$(id -g)" \
		-v "$$(pwd)/v2/api:/app/output" \
		-v "$$(pwd)/v2/openapi:/app/openapi:ro" \
		mcr.microsoft.com/openapi/kiota:1.31.1 generate --exclude-backward-compatible --language go --openapi /app/openapi/spec.json --namespace-name github.com/hashicorp/go-tfe/v2/api

spec_internal:
	./scripts/spec_internal.sh

api_internal: spec_internal
	docker run --rm \
		--user "$$(id -u):$$(id -g)" \
		-v "$$(pwd)/v2/api:/app/output" \
		-v "$$(pwd)/v2/openapi:/app/openapi:ro" \
		mcr.microsoft.com/openapi/kiota:1.31.1 generate --exclude-backward-compatible --language go --openapi /app/openapi/spec.json --namespace-name github.com/hashicorp/go-tfe/v2/internal/api