# This rule runs the resource scaffolding script
# $1 is the name of the resource to generate
check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = \
    $(if $(value $1),, \
      $(error Undefined $1$(if $2, ($2))))

.PHONY: vet fmt lint test mocks envvars generate

generate:
	$(call check_defined, RESOURCE)
	@cd ./scripts/generate_resource; \
	go mod tidy; \
	go run . $(RESOURCE) ;

fmt:

vet:
	go vet

fmt:
	go fmt ./...

lint:
	golangci-lint run .

test:
	go test ./... $(TESTARGS) -tags=integration -timeout=30m

mocks:
	$(call check_defined, FILENAME)
	@echo "mockgen -source=$(FILENAME) -destination=mocks/$(FILENAME) -package=mocks" >> generate_mocks.sh
	./generate_mocks.sh

envvars:
	./scripts/setup-test-envvars.sh

