ifeq (/,${HOME})
GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache/
else
GOLANGCI_LINT_CACHE=${HOME}/.cache/golangci-lint
endif
GOLANGCI_LINT ?= GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) go run github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: build
build: generate
	CGO_ENABLED=0 go build -o bin/nitric ./pkg/cmd/

.PHONY: generate
generate:
	@go run github.com/golang/mock/mockgen github.com/nitrictech/cli/pkg/containerengine ContainerEngine > mocks/mock_containerengine/mock_containerengine.go
	@go run github.com/golang/mock/mockgen github.com/nitrictech/cli/pkg/utils GetterClient > mocks/mock_utils/mock_getter.go
	@go run ./hack/modversion/main.go > pkg/stack/membraneversion.txt

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
