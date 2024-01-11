package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type RuntimeConfiguration struct {
	// Template dockerfile to use as the base for the custom runtime
	Dockerfile string
	// Additional args to pass to the custom runtime
	Args map[string]string
}

type ServiceConfiguration struct {
	// This is the string version
	Match string `yaml:"match"`

	// This is the custom runtime version (is custom if not nil, we auto-detect a standard language runtime)
	Runtime string `yaml:"runtime"`

	// This allows specifying a particular service type (e.g. "Job"), this is optional and custom service types can be defined for each stack
	Type string `yaml:"type"`
}

type ProjectConfiguration struct {
	Name      string                          `yaml:"name"`
	Directory string                          `yaml:"-"`
	Services  []ServiceConfiguration          `yaml:"services"`
	Runtimes  map[string]RuntimeConfiguration `yaml:"runtimes,omitempty"`
}

const defaultNitricYamlPath = "./nitric.yaml"

func (p ProjectConfiguration) ToFile(fs afero.Fs, filepath string) error {
	nitricYamlPath := defaultNitricYamlPath

	if filepath != "" {
		nitricYamlPath = filepath
	}

	projectBytes, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	if err = afero.WriteFile(fs, nitricYamlPath, projectBytes, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func ConfigurationFromFile(fs afero.Fs, filePath string) (*ProjectConfiguration, error) {
	if filePath == "" {
		filePath = defaultNitricYamlPath
	}

	absProjectDir, err := filepath.Abs(filepath.Dir(filePath))
	if err != nil {
		return nil, err
	}

	// Check if the path is a directory
	info, err := fs.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("nitric.yaml not found in %s. A nitric project is required to load configuration", absProjectDir)
		}
		return nil, err
	}

	if info.IsDir() {
		return nil, fmt.Errorf("nitric.yaml path %s is a directory", filePath)
	}

	projectFileContents, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read nitric.yaml: %s", err)
	}

	// TODO: Implement v0 yaml detection and provide link to the upgrade guide

	projectConfig := &ProjectConfiguration{}

	if err := yaml.Unmarshal(projectFileContents, projectConfig); err != nil {
		return nil, fmt.Errorf("unable to parse nitric.yaml: %s", err)
	}

	projectConfig.Directory = filepath.Dir(filePath)

	return projectConfig, nil
}
