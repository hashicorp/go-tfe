.PHONY: vet fmt lint test mocks envvars generate

vet:
	go vet

fmt:
	gofmt -s -l -w .

fmtcheck:
	./scripts/gofmtcheck.sh

lint:
	golangci-lint run .

test:
	cd v2 && go test ./... $(TESTARGS)

openapi/spec.json:
	mkdir -p v2/openapi
	mkdir -p v2/api
	cp ../atlas/openapi/bundled/hcpt_v2_public_beta.json v2/openapi/spec.json

api: openapi/spec.json
	docker run -v ./v2/api:a/app/output -v ./v2/openapi/spec.json:/app/openapi.json mcr.microsoft.com/openapi/kiota:1.31.1 generate --exclude-backward-compatible --language go --openapi openapi.json --namespace-name github.com/hashicorp/go-tfe/v2/api
