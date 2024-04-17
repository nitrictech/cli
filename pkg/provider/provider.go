package provider

import (
	"fmt"
	"slices"
	"strings"

	"github.com/nitrictech/cli/pkg/preview"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/spf13/afero"
)

type Provider interface {
	Install() error
	Start(opts *StartOptions) (string, error)
	Stop() error
}

type StartOptions struct {
	Env    map[string]string
	StdOut chan<- string
	StdErr chan<- string
}

const nitricOrg = "nitric"

// NewProvider - Returns a new provider instance based on the given providerId string
// The providerId string is in the form of <org-name>/<provider-name>@<version>
func NewProvider(providerId string, project *project.Project, fs afero.Fs) (Provider, error) {
	if strings.HasPrefix(providerId, "docker://") {
		if !slices.Contains(project.Preview, preview.Feature_DockerProviders) {
			return nil, fmt.Errorf("your stack specifies %s as the provider, docker providers are not enabled for this project. See https://nitric.io/docs/reference/providers/install/docker", providerId)
		}
		// remove the prefix and return a new image provider with the URI
		dockerUri := strings.Replace(providerId, "docker://", "", 1)
		return &ProviderImage{
			imageName: dockerUri,
		}, nil
	}

	// Default to standard provider
	provider, err := NewStandardProvider(providerId, fs)
	if err != nil {
		return nil, err
	}

	if provider.organization == nitricOrg {
		// v0 providers are not supported, still permit the 'development' version 0.0.1
		if strings.HasPrefix(provider.version, "0.") && provider.version != "0.0.1" {
			return nil, fmt.Errorf("nitric providers prior to version 1.0.0 are not supported by this version of the CLI")
		}
	}

	return provider, nil
}
