ifeq (/,${HOME})
    GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache/
else
    GOLANGCI_LINT_CACHE=${HOME}/.cache/golangci-lint
endif
GOLANGCI_LINT ?= GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) go run github.com/golangci/golangci-lint/cmd/golangci-lint

ifeq ($(OS), Windows_NT)
    OS=Windows
    BUILD_ENV=CGO_ENABLED=0
    EXECUTABLE_EXT=.exe
else
    OS=$(shell uname -s)
    BUILD_ENV=CGO_ENABLED=0
    EXECUTABLE_EXT=
endif

# See pkg/cmd/version.go for details
SOURCE_GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_VERSION ?= $(shell git describe --tags --abbrev=40 --dirty)
VERSION_URI = "github.com/nitrictech/cli/pkg/version"
export LDFLAGS="-X $(VERSION_URI).Version=${BUILD_VERSION} \
                -X $(VERSION_URI).Commit=${SOURCE_GIT_COMMIT} \
                -X $(VERSION_URI).BuildTime=$(shell date +%Y-%m-%dT%H:%M:%S%z)"

.PHONY: build
build: generate build-cli

.PHONY: build-cli
build-cli:
	$(BUILD_ENV) go build -ldflags $(LDFLAGS) -o bin/nitric$(EXECUTABLE_EXT) ./main.go

.PHONY: build-dashboard
build-dashboard: 
	yarn --cwd ./pkg/dashboard/frontend install
	yarn --cwd ./pkg/dashboard/frontend build

.PHONY: generate
generate: build-dashboard
	# @go run github.com/golang/mock/mockgen github.com/nitrictech/cli/pkg/containerengine ContainerEngine > mocks/mock_containerengine/mock_containerengine.go
	@go run github.com/golang/mock/mockgen github.com/nitrictech/cli/pkg/project/templates GetterClient > mocks/mock_utils/mock_getter.go
	@go run ./hack/readmegen/ README.md

.PHONY: fmt
fmt:
	$(GOLANGCI_LINT) run --fix

.PHONY: lint
lint: build-dashboard
	$(GOLANGCI_LINT) run --timeout=10m

.PHONY: test
test: build-dashboard
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
	go run ./hack/github_release open-telemetry opentelemetry-collector-releases > pkg/project/otel-collector-version.txt
	go get -u $$(go run github.com/nitrictech/go-mods-direct)
	go mod tidy
