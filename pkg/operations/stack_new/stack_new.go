package stack_new

import (
	"github.com/AlecAivazis/survey/v2"

	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/utils"
)

func Run() error {
	name := ""

	err := survey.AskOne(&survey.Input{
		Message: "What do you want to call your new stack?",
	}, &name)
	if err != nil {
		return err
	}

	pName := ""

	err = survey.AskOne(&survey.Select{
		Message: "Which Cloud do you wish to deploy to?",
		Default: types.Aws,
		Options: types.Providers,
	}, &pName)
	if err != nil {
		return err
	}

	pc, err := project.ConfigFromProjectPath("")
	if err != nil {
		return err
	}

	cc, err := codeconfig.New(project.New(pc.BaseConfig), map[string]string{})
	utils.CheckErr(err)

	prov, err := provider.NewProvider(cc, name, pName, map[string]string{}, &types.ProviderOpts{})
	if err != nil {
		return err
	}

	return prov.AskAndSave()
}
