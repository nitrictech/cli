build:
	@echo Building the Nitric CLI
	@CGO_ENABLED=0 go build -o bin/nitric ./pkg/cmd/
