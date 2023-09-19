package stack_update

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/pterm/pterm"

	"github.com/nitrictech/cli/pkg/build"
	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/tasklet"
	"github.com/nitrictech/cli/pkg/utils"
)

func Run(envFile string, s *stack.Config, force bool) {
	config, err := project.ConfigFromProjectPath("")
	utils.CheckErr(err)

	proj, err := project.FromConfig(config)
	utils.CheckErr(err)

	log.SetOutput(output.NewPtermWriter(pterm.Debug))
	log.SetFlags(0)

	envFiles := utils.FilesExisting(".env", ".env.production", envFile)
	envMap := map[string]string{}
	if len(envFiles) > 0 {
		envMap, err = godotenv.Read(envFiles...)
		utils.CheckErr(err)
	}

	// build base images on updates
	createBaseImage := tasklet.Runner{
		StartMsg: "Building Images",
		Runner: func(_ output.Progress) error {
			return build.BuildBaseImages(proj)
		},
		StopMsg: "Images Built",
	}
	tasklet.MustRun(createBaseImage, tasklet.Opts{})

	cc, err := codeconfig.New(proj, envMap)
	utils.CheckErr(err)

	codeAsConfig := tasklet.Runner{
		StartMsg: "Gathering configuration from code..",
		Runner: func(_ output.Progress) error {
			return cc.Collect()
		},
		StopMsg: "Configuration gathered",
	}
	tasklet.MustRun(codeAsConfig, tasklet.Opts{})

	p, err := provider.ProviderFromFile(cc, s.Name, s.Provider, envMap, &types.ProviderOpts{Force: force})
	utils.CheckErr(err)

	d := &types.Deployment{}
	deploy := tasklet.Runner{
		StartMsg: "Deploying..",
		Runner: func(progress output.Progress) error {
			d, err = p.Up(progress)

			return err
		},
		StopMsg: "Stack",
	}
	tasklet.MustRun(deploy, tasklet.Opts{SuccessPrefix: "Deployed"})

	// Print callable APIs if any were deployed
	if len(d.ApiEndpoints) > 0 {
		rows := [][]string{{"API", "Endpoint"}}
		for k, v := range d.ApiEndpoints {
			rows = append(rows, []string{k, v})
		}
		_ = pterm.DefaultTable.WithBoxed().WithData(rows).Render()
	}
}
