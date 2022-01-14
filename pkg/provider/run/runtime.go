package run

import (
	"fmt"
	"path/filepath"

	"github.com/docker/docker/api/types/strslice"
	"github.com/nitrictech/newcli/pkg/build"
	"github.com/spf13/cobra"
)

type Runtime string

const (
	RuntimeTypescript Runtime = "ts"
	RuntimeJavascript Runtime = "js"
)

func devImageNameForRuntime(runtime Runtime) string {
	return fmt.Sprintf("nitric-%s-dev", runtime)
}

type LaunchOpts struct {
	Entrypoint []string
	Cmd        []string
}

func launchOptsForFunction(f *Function) (LaunchOpts, error) {
	switch f.runtime {
	// Javascript will re-use typescript runtime
	case RuntimeJavascript:
	case RuntimeTypescript:
		return LaunchOpts{
			Entrypoint: strslice.StrSlice{"nodemon"},
			Cmd:        strslice.StrSlice{"--watch", "/app/**", "--ext", "ts,js,json", "--exec", "ts-node -T " + "/app/" + f.handler},
		}, nil
	}

	return LaunchOpts{}, fmt.Errorf("unsupported runtime")
}

func CreateBaseDevForFunctions(funcs []*Function) error {
	ctx, _ := filepath.Abs(".")
	if err := build.CreateBaseDev(ctx, map[string]string{
		"ts": "nitric-ts-dev",
	}); err != nil {
		cobra.CheckErr(err)
	}

	imageBuilds := make(map[string]string)

	for _, f := range funcs {
		switch f.runtime {
		// Javascript will re-use typescript runtime
		case RuntimeJavascript:
		case RuntimeTypescript:
			imageBuilds[string(RuntimeTypescript)] = devImageNameForRuntime(RuntimeTypescript)
		}
	}

	// Currently the file context does not matter (base runtime images should not copy files)
	return build.CreateBaseDev(".", imageBuilds)
}
