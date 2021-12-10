package functiondockerfile

import (
	"io"
	"strings"

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
	"github.com/nitrictech/newcli/pkg/stack"
)

func pythonGenerator(f *stack.Function, version, provider string, w io.Writer) error {
	con, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   "python:3.7-slim",
		Ignore: []string{"__pycache__/", "*.py[cod]", "*$py.class"},
	})
	if err != nil {
		return err
	}

	con.Run(dockerfile.RunOptions{Command: []string{"pip", "install", "--upgrade", "pip"}})
	con.Config(dockerfile.ConfigOptions{
		WorkingDir: "/",
	})
	con.Copy(dockerfile.CopyOptions{Src: "requirements.txt", Dest: "requirements.txt"})
	con.Run(dockerfile.RunOptions{Command: []string{"pip", "install", "--no-cache-dir", "-r", "requirements.txt"}})
	con.Copy(dockerfile.CopyOptions{Src: ".", Dest: "."})

	withMembrane(con, version, provider)

	con.Config(dockerfile.ConfigOptions{
		Env: map[string]string{
			"PYTHONPATH": "/app/:${PYTHONPATH}",
		},
		Ports: []int32{9001},
		Cmd:   []string{"python", f.Handler},
	})
	_, err = w.Write([]byte(strings.Join(con.Lines(), "\n")))
	return err
}
