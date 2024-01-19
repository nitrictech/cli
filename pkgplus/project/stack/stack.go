package stack

import (
	_ "embed"
	"fmt"
	"os"
	"regexp"

	"github.com/nitrictech/cli/pkgplus/paths"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type StackConfig[T any] struct {
	Name     string `yaml:-`
	Provider string `yaml:"provider"`
	Config   T      `yaml:",inline"`
}

//go:embed aws.config.yaml
var awsConfigTemplate string

//go:embed azure.config.yaml
var azureConfigTemplate string

//go:embed gcp.config.yaml
var gcpConfigTemplate string

var fileNameRegex = regexp.MustCompile(`(?i)^nitric\.(\S+)\.ya?ml$`)

func IsValidFileName(stackName string) bool {
	return fileNameRegex.MatchString(stackName)
}

func NewStackFile(fs afero.Fs, providerName string, stackName string, dir string) (string, error) {
	if dir == "" {
		dir = "./"
	}

	var template string = ""
	switch providerName {
	case "aws":
		template = awsConfigTemplate
	case "gcp":
		template = gcpConfigTemplate
	case "azure":
		template = azureConfigTemplate
	}

	fileName := StackFileName(stackName)

	if !IsValidFileName(fileName) {
		return "", fmt.Errorf("invalid stack name %s", stackName)
	}

	stackFilePath := paths.Join(dir, fileName)
	relativePath, _ := paths.Rel(".", stackFilePath)

	return fmt.Sprintf(".%s%s", string(os.PathSeparator), relativePath), afero.WriteFile(fs, stackFilePath, []byte(template), os.ModePerm)
}

// StackFileName returns the stack file name for a given stack name
func StackFileName(stackName string) string {
	return fmt.Sprintf("nitric.%s.yaml", stackName)
}

// ConfigFromName returns a stack configuration from a given stack name
func ConfigFromName[T any](fs afero.Fs, stackName string) (*StackConfig[T], error) {
	stackFile := StackFileName(stackName)
	if !IsValidFileName(stackFile) {
		return nil, fmt.Errorf("invalid stack name %s", stackName)
	}
	return configFromFile[T](fs, paths.Join("./", stackFile))
}

// GetAllStackFiles returns a list of all stack files in the current directory
func GetAllStackFiles(fs afero.Fs) ([]string, error) {
	return paths.Glob(fs, "nitric.*.yaml")
}

// GetStackNameFromFileName returns the stack name from a given stack file name
// e.g. nitric.aws.yaml -> aws
func GetStackNameFromFileName(fileName string) string {
	matches := fileNameRegex.FindStringSubmatch(fileName)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ConfigFromFile returns a stack configuration from a given stack file
func configFromFile[T any](fs afero.Fs, filePath string) (*StackConfig[T], error) {
	stackFileContents, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return nil, err
	}

	stackConfig := &StackConfig[T]{}

	if err := yaml.Unmarshal(stackFileContents, stackConfig); err != nil {
		return nil, err
	}

	stackConfig.Name = GetStackNameFromFileName(filePath)
	if stackConfig.Name == "" {
		return nil, fmt.Errorf("no stack name found in stack file pattern")
	}

	return stackConfig, nil
}
