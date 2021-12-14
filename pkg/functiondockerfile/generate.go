package functiondockerfile

import (
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
	"github.com/nitrictech/newcli/pkg/stack"
)

type FunctionDockerfile interface {
	Generate(io.Writer) error
}

func withMembrane(con dockerfile.ContainerState, version, provider string) {
	fetchFrom := fmt.Sprintf("https://github.com/nitrictech/nitric/releases/download/%s/membrane-%s", version, provider)
	if version == "latest" {
		fetchFrom = fmt.Sprintf("https://github.com/nitrictech/nitric/releases/%s/download/membrane-%s", version, provider)
	}
	con.Add(dockerfile.AddOptions{Src: fetchFrom, Dest: "/usr/local/bin/membrane"})
	con.Run(dockerfile.RunOptions{Command: []string{"chmod", "+x-rw", "/usr/local/bin/membrane"}})
	con.Config(dockerfile.ConfigOptions{
		Entrypoint: []string{"/usr/local/bin/membrane"},
	})
}

func Generate(f *stack.Function, version, provider string, fwriter io.Writer) error {
	switch path.Ext(f.Handler) {
	case ".js":
		return javascriptGenerator(f, version, provider, fwriter)
	case ".ts":
		return typescriptGenerator(f, version, provider, fwriter)
	case ".go":
		return golangGenerator(f, version, provider, fwriter)
	case ".py":
		return pythonGenerator(f, version, provider, fwriter)
	case ".jar":
		return javaGenerator(f, version, provider, fwriter)
	}
	return errors.New("could not build dockerfile from " + f.Handler + ", extension not supported")
}
