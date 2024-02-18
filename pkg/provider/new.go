package provider

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/nitrictech/cli/pkg/paths"
)

type Provider struct {
	organization string
	name         string
	version      string
}

const semverRegex = `@(latest|(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*)))?`

// Provider format <org>/<provider>@<semver>
const providerIdRegex = `\w+\/\w+` + semverRegex

func providerIdSeparators(r rune) bool {
	const versionSeparator = '@'
	const orgSeparator = '/'
	return r == versionSeparator || r == orgSeparator
}

func providerFromId(providerId string) (*Provider, error) {
	match, err := regexp.MatchString(providerIdRegex, providerId)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred parsing provider ID %s (%w)", providerId, err)
	}

	if !match {
		return nil, fmt.Errorf("invalid provider format %s, valid example is nitric/aws@1.2.3", providerId)
	}

	providerParts := strings.FieldsFunc(providerId, providerIdSeparators)

	return &Provider{
		organization: providerParts[0],
		name:         providerParts[1],
		version:      providerParts[2],
	}, nil
}

const nitricOrg = "nitric"

const (
	nitricAwsProvider   = "aws"
	nitricGcpProvider   = "gcp"
	nitricAzureProvider = "azure"
)

func providerFilePath(prov *Provider) string {
	provDir := paths.NitricProviderDir()
	os := runtime.GOOS

	if os == "windows" {
		return filepath.Join(provDir, prov.organization, fmt.Sprintf("%s-%s%s", prov.name, prov.version, ".exe"))
	}

	return filepath.Join(provDir, prov.organization, fmt.Sprintf("%s-%s", prov.name, prov.version))
}

// NewProvider - Returns a new provider instance based on the given providerId string
// The providerId string is in the form of <org-name>/<provider-name>@<version>
func NewProvider(providerId string) (*Provider, error) {
	provider, err := providerFromId(providerId)
	if err != nil {
		return nil, err
	}

	if provider.organization == "nitric" {
	}

	return provider, nil
}

// func NewDeploymentEngine(provider *Provider) (DeploymentEngine, error) {
// 	baseNitricDeployment := &nitricDeployment{binaryRemoteDeployment: baseBinaryDeployment}

// 	// Format provider file location
// 	providerFilePath := providerFilePath(provider)
// 	if provider.organization == nitricOrg {
// 		// attempt to install
// 		providerFile, err = ensureProviderExists(provider)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	return NewProviderExecutable(providerFilePath)
// }
