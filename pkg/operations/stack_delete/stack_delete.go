package stack_delete

import (
	"log"

	"github.com/pterm/pterm"

	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/tasklet"
	"github.com/nitrictech/cli/pkg/utils"
)

func Run() {
	s, err := stack.ConfigFromOptions()
	utils.CheckErr(err)

	log.SetOutput(output.NewPtermWriter(pterm.Debug))
	log.SetFlags(0)

	config, err := project.ConfigFromProjectPath("")
	utils.CheckErr(err)

	proj, err := project.FromConfig(config)
	utils.CheckErr(err)

	cc, err := codeconfig.New(proj, map[string]string{})
	utils.CheckErr(err)

	p, err := provider.ProviderFromFile(cc, s.Name, s.Provider, map[string]string{}, &types.ProviderOpts{Force: true})
	utils.CheckErr(err)

	deploy := tasklet.Runner{
		StartMsg: "Deleting..",
		Runner: func(progress output.Progress) error {
			_, err := p.Down(progress)

			return err
		},
		StopMsg: "Stack",
	}
	tasklet.MustRun(deploy, tasklet.Opts{
		SuccessPrefix: "Deleted",
	})
}
