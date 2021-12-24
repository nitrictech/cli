module github.com/nitrictech/newcli

go 1.16

require (
	github.com/AlecAivazis/survey/v2 v2.3.2
	github.com/containers/buildah v1.23.1
	github.com/containers/image/v5 v5.17.1-0.20211207161909-6f3c8453e1a7 // indirect
	github.com/containers/podman/v3 v3.4.4
	github.com/cri-o/ocicni v0.2.1-0.20210621164014-d0acc7862283
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/docker v20.10.11+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/fatih/color v1.13.0
	github.com/golangci/golangci-lint v1.43.0
	github.com/google/go-github/v41 v41.0.0
	github.com/hashicorp/consul/sdk v0.8.0
	github.com/jhoonb/archivex v0.0.0-20201016144719-6a343cdae81d
	github.com/mitchellh/mapstructure v1.4.2
	github.com/nitrictech/boxygen v0.0.1-rc.7.0.20211212231606-62c668408f91
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.9.0
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2-0.20211123152302-43a7dee1ec31
	github.com/rootless-containers/rootlesskit => github.com/rootless-containers/rootlesskit v0.14.6
)
