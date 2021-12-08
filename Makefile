ifeq (/,${HOME})
GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache/
else
GOLANGCI_LINT_CACHE=${HOME}/.cache/golangci-lint
endif
GOLANGCI_LINT ?= GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) go run github.com/golangci/golangci-lint/cmd/golangci-lint

build:
	@echo Building the Nitric CLI
	@CGO_ENABLED=0 go build -o bin/nitric ./pkg/cmd/

fmt:
	$(GOLANGCI_LINT) run --fix

lint:
	$(GOLANGCI_LINT) run

test:
	go test -v ./pkg/...
