package version

import (
	"context"
	"fmt"
	"runtime"

	"github.com/nitrictech/cli/pkg/utils"
)

func Run(ctx context.Context) {
	fmt.Printf("Nitric CLI: %s\n", utils.Version)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Git commit: %s\n", utils.Commit)
	fmt.Printf("Build time: %s\n", utils.BuildTime)
}
