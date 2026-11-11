.PHONY: vet fmt lint test spec api api_internal spec_internal

vet:
	cd v2 && go vet

fmt:
	cd v2 && gofmt -s -l -w .

fmtcheck:
	./scripts/gofmtcheck.sh

lint:
	cd v2 && golangci-lint run

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
		mcr.microsoft.com/openapi/kiota:1.38.0 generate --exclude-backward-compatible --backing-store --language go --openapi /app/openapi/spec.json --namespace-name github.com/hashicorp/go-tfe/v2/api
	go run ./scripts/fix_kiota_backing_store.go v2/api

spec_internal:
	./scripts/spec_internal.sh

api_internal: spec_internal
	docker run --rm \
		--user "$$(id -u):$$(id -g)" \
		-v "$$(pwd)/v2/api:/app/output" \
		-v "$$(pwd)/v2/openapi:/app/openapi:ro" \
		mcr.microsoft.com/openapi/kiota:1.38.0 generate --exclude-backward-compatible --backing-store --language go --openapi /app/openapi/spec.json --namespace-name github.com/hashicorp/go-tfe/v2/api
	go run ./scripts/fix_kiota_backing_store.go v2/api
