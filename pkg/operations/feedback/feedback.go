package feedback

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/cli/pkg/ghissue"
	"github.com/nitrictech/cli/pkg/utils"
)

func Run(ctx context.Context) {
	answers := struct {
		Repo  string
		Kind  string
		Title string
		Body  string
	}{}

	d := ghissue.Gather()

	diag, err := yaml.Marshal(d)
	utils.CheckErr(err)

	qs := []*survey.Question{
		{
			Name: "repo",
			Prompt: &survey.Select{
				Message: "What is the name of the repo?",
				Options: []string{"cli", "nitric", "docs", "apis", "node-sdk", "go-sdk"},
			},
		},
		{
			Name: "kind",
			Prompt: &survey.Select{
				Message: "What kind of feedback do you want to give?",
				Options: []string{"bug", "feature-request", "question"},
			},
		},
		{
			Name: "title",
			Prompt: &survey.Input{
				Message: "How would you like to title your feedback?",
			},
		},
		{
			Name: "body",
			Prompt: &survey.Editor{
				Message:       "Please write your feedback",
				Default:       string(diag),
				HideDefault:   true,
				AppendDefault: true,
			},
		},
	}
	err = survey.Ask(qs, &answers)
	utils.CheckErr(err)

	pterm.Info.Println("Please create a github issue by clicking on the link below")
	fmt.Println(ghissue.IssueLink(answers.Repo, answers.Kind, answers.Title, answers.Body))
}
