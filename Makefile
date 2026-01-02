.PHONY: vet fmt fmtchewck lint test mocks envvars api openapi
vet:
	go vet

fmt:
	gofmt -s -l -w .

fmtcheck:
	./scripts/gofmtcheck.sh

lint:
	golangci-lint run .

test:
	go test ./... $(TESTARGS) -timeout=30m

# Make target to generate mocks for specified FILENAME
mocks: check-filename
	@echo "mockgen -source=$(FILENAME) -destination=mocks/$(subst .go,_mocks.go,$(FILENAME)) -package=mocks" >> generate_mocks.sh
	./generate_mocks.sh

envvars:
	./scripts/setup-test-envvars.sh

openapi:
	mkdir -p openapi
	mkdir -p api
	cp ../atlas/openapi/bundled/hcpt_v2.json openapi/spec.json

api: openapi
	docker run -v ./api:/app/output -v ./openapi/spec.json:/app/openapi.json mcr.microsoft.com/openapi/kiota:1.25.1 generate --language go --openapi openapi.json --namespace-name github.com/hashicorp/go-tfe/api
