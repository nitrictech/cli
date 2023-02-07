package remote

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/utils"
)

// Provides a binary remote provider type
type binaryRemoteDeployment struct {
	providerPath string
	*remoteDeployment
}

func (p *binaryRemoteDeployment) startProcess() (*os.Process, error) {
	cmd := exec.Command(p.providerPath)

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd.Process, nil
}

func (p *binaryRemoteDeployment) Up(log output.Progress) (*types.Deployment, error) {
	// start the provider command
	process, err := p.startProcess()
	if err != nil {
		return nil, err
	}
	defer process.Kill() //nolint:errcheck

	return p.remoteDeployment.Up(log)
}

func (p *binaryRemoteDeployment) Down(log output.Progress) (*types.Summary, error) {
	// start the provider command
	process, err := p.startProcess()
	if err != nil {
		return nil, err
	}

	defer process.Kill() //nolint:errcheck

	return p.remoteDeployment.Down(log)
}

func isExecAny(mode os.FileMode) bool {
	return mode&0111 != 0
}

func providerFilePath(prov *provider) string {
	provDir := utils.NitricProviderDir()

	return path.Join(provDir, prov.org, fmt.Sprintf("%s-%s", prov.name, prov.version))
}

func NewBinaryRemoteDeployment(cfc types.ConfigFromCode, sc *StackConfig, prov *provider, envMap map[string]string, opts *types.ProviderOpts) (types.Provider, error) {
	// See if the binary exists in NITRIC_HOME/providers
	providerPath := providerFilePath(prov)

	fi, err := os.Stat(providerPath)
	if err != nil {
		return nil, err
	}

	// Ensure the file is executable
	if !isExecAny(fi.Mode()) {
		return nil, fmt.Errorf("provider exists but is not executable")
	}

	// return a valid binary deployment
	return &binaryRemoteDeployment{
		providerPath: providerPath,
		remoteDeployment: &remoteDeployment{
			cfc: cfc,
			sfc: sc,
		},
	}, nil
}
