package provider

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/hashicorp/go-getter"
	"github.com/nitrictech/cli/pkg/iox"
	"github.com/nitrictech/cli/pkg/netx"
	"github.com/nitrictech/cli/pkg/paths"
	"github.com/spf13/afero"
)

const semverRegex = `@(latest|(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*)))?`

// Provider format <org>/<provider>@<semver>
const providerIdRegex = `\w+\/\w+` + semverRegex

func providerIdSeparators(r rune) bool {
	const versionSeparator = '@'
	const orgSeparator = '/'
	return r == versionSeparator || r == orgSeparator
}

type StandardProvider struct {
	organization string
	name         string
	version      string
	fs           afero.Fs
	process      *os.Process
}

var _ Provider = (*StandardProvider)(nil)

// Gets a default provider string and translates it into a file name that can be retrieved from
// our github releases
func (prov *StandardProvider) providerFileName() string {
	// Get the OS name
	os := runtime.GOOS
	platform := runtime.GOARCH

	// tarballs are the default archive type
	archive := "tar.gz"
	if os == "windows" {
		// We use zips for windows
		archive = "zip"
	}

	if platform == "amd64" {
		platform = "x86_64"
	}

	// Return the archive uri in the form of
	// {PROVIDER}_{OS}_{PLATFORM}.{ARCHIVE}
	// e.g. gcp_linux_x86_64.tar.gz
	return strings.ToLower(fmt.Sprintf("%s_%s_%s.%s", prov.name, os, platform, archive))
}

func (sp *StandardProvider) binaryFilePath() string {
	provDir := paths.NitricProviderDir()
	os := runtime.GOOS

	if os == "windows" {
		return filepath.Join(provDir, sp.organization, fmt.Sprintf("%s-%s%s", sp.name, sp.version, ".exe"))
	}

	return filepath.Join(provDir, sp.organization, fmt.Sprintf("%s-%s", sp.name, sp.version))
}

func (prov *StandardProvider) defaultDownloadUri() string {
	fileName := prov.providerFileName()

	if prov.version == "latest" {
		return fmt.Sprintf("https://github.com/nitrictech/nitric/releases/latest/download/%s", fileName)
	}

	return fmt.Sprintf("https://github.com/nitrictech/nitric/releases/download/v%s/%s", prov.version, fileName)
}

func (sp *StandardProvider) Install() error {
	// Check to see if the provider already exists
	provFile := sp.binaryFilePath()

	// Check if the provider we're after actually exists already
	_, err := sp.fs.Stat(provFile)

	if err != nil && sp.organization == nitricOrg {
		// If the provider is apart of the nitric org attempt to download it from the core nitric releases
		if sp.organization == nitricOrg {
			if err := getter.GetFile(provFile, sp.defaultDownloadUri()); err != nil {
				return fmt.Errorf("error downloading file %s (%w)", sp.defaultDownloadUri(), err)
			}
		} else {
			// Not a nitric release so should be installed manually
			// TODO: Make CLI assistant method for getting third-part provider releases
			// nitric provider install custom provider --url "https://github.com/my-org/my-project/releases"
			return fmt.Errorf("could not locate provider at %s, please check that it exists and is executable", provFile)
		}
	}

	return nil
}

func (sp *StandardProvider) Start(opts *StartOptions) (string, error) {
	cmd := exec.Command(sp.binaryFilePath())

	lis, err := netx.GetNextListener()
	if err != nil {
		return "", err
	}

	tcpAddr := lis.Addr().(*net.TCPAddr)

	// Set a random available port
	address := lis.Addr().String()

	// TODO: consider prefixing with NITRIC_ to avoid collisions
	containerEnv := map[string]string{
		"PORT": fmt.Sprint(tcpAddr.Port),
	}

	for k, v := range opts.Env {
		containerEnv[k] = v
	}

	if len(containerEnv) > 0 {
		env := os.Environ()

		for k, v := range containerEnv {
			env = append(env, k+"="+v)
		}

		cmd.Env = env
	}

	err = lis.Close()
	if err != nil {
		return "", err
	}

	cmd.Stderr = io.Discard
	cmd.Stdout = io.Discard

	if opts.StdErr != nil {
		cmd.Stderr = iox.NewChannelWriter(opts.StdErr)
	}

	if opts.StdOut != nil {
		cmd.Stdout = iox.NewChannelWriter(opts.StdOut)
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	sp.process = cmd.Process

	return address, nil
}

func (sp *StandardProvider) Stop() error {
	if sp.process != nil {
		err := sp.process.Kill()
		if err != nil {
			return fmt.Errorf("failed to stop provider: %w", err)
		}
	}

	return nil
}

func NewStandardProvider(providerId string, fs afero.Fs) (*StandardProvider, error) {
	match, err := regexp.MatchString(providerIdRegex, providerId)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred parsing provider ID %s (%w)", providerId, err)
	}

	if !match {
		return nil, fmt.Errorf("invalid provider format %s, valid example is nitric/aws@1.2.3", providerId)
	}

	providerParts := strings.FieldsFunc(providerId, providerIdSeparators)

	return &StandardProvider{
		organization: providerParts[0],
		name:         providerParts[1],
		version:      providerParts[2],
		fs:           fs,
	}, nil
}
