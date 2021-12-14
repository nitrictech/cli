package functiondockerfile

import (
	"io"
	"strings"

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
	"github.com/nitrictech/newcli/pkg/stack"
)

func typescriptGenerator(f *stack.Function, version, provider string, w io.Writer) error {
	con, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   "node:alpine",
		Ignore: []string{"node_modules/", ".nitric/", ".git/", ".idea/"},
	})
	if err != nil {
		return err
	}
	withMembrane(con, version, provider)

	con.Run(dockerfile.RunOptions{Command: []string{"yarn", "global", "add", "typescript"}})
	con.Run(dockerfile.RunOptions{Command: []string{"yarn", "global", "add", "ts-node"}})
	con.Copy(dockerfile.CopyOptions{Src: "package.json *.lock *-lock.json", Dest: "/"})
	con.Run(dockerfile.RunOptions{Command: []string{"yarn", "import", "||", "echo", "Lockfile already exists"}})
	con.Run(dockerfile.RunOptions{Command: []string{
		"set", "-ex;",
		"yarn", "install", "--production", "--frozen-lockfile", "--cache-folder", "/tmp/.cache;",
		"rm", "-rf", "/tmp/.cache;"}})
	con.Config(dockerfile.ConfigOptions{
		Cmd: []string{"node", f.Handler},
	})
	_, err = w.Write([]byte(strings.Join(con.Lines(), "\n")))
	return err
}
