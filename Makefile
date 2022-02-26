ifeq (/,${HOME})
GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache/
else
GOLANGCI_LINT_CACHE=${HOME}/.cache/golangci-lint
endif
GOLANGCI_LINT ?= GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) go run github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: build
build: generate
	CGO_ENABLED=0 go build -o bin/nitric ./pkg/cmd/

.PHONY: build-windows
build-windows: generate
	go build -o bin/nitric.exe ./pkg/cmd/	

.PHONY: generate
generate:
	@go run github.com/golang/mock/mockgen github.com/nitrictech/cli/pkg/containerengine ContainerEngine > mocks/mock_containerengine/mock_containerengine.go
	@go run github.com/golang/mock/mockgen github.com/nitrictech/cli/pkg/utils GetterClient > mocks/mock_utils/mock_getter.go
	@go run ./hack/modversion "github.com/nitrictech/nitric" > pkg/stack/membraneversion.txt
	@go run ./hack/modversion "github.com/pulumi/pulumi-azuread/" > pkg/provider/pulumi/azure/pulumi-azuread-version.txt
	@go run ./hack/modversion "github.com/pulumi/pulumi-azure/" > pkg/provider/pulumi/azure/pulumi-azure-version.txt
	@go run ./hack/modversion "github.com/pulumi/pulumi-azure-native/" > pkg/provider/pulumi/azure/pulumi-azure-native-version.txt
	@go run ./hack/modversion "github.com/pulumi/pulumi-aws/" > pkg/provider/pulumi/aws/pulumi-aws-version.txt


.PHONY: fmt
fmt:
	$(GOLANGCI_LINT) run --fix

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run

.PHONY: test
test:
	go test ./pkg/...

test-coverage:
	@go test -cover -outputdir=./ -coverprofile=all.coverprofile ./pkg/...

coverage-html: test-coverage
	@go tool cover -html=all.coverprofile

generate_check: generate fmt
	@if [ -n "$$(git ls-files -m)" ]; then \
        echo "'make generate' requires you to check in the following files:"; \
		git ls-files -m ; \
		exit 1 ; \
    fi

.PHONY: check
check: lint test generate_check

.PHONY: go-mod-update
go-mod-update:
	go get -u $$(go run ./hack/allmods)
	go mod tidy
