package teax

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// FullViewProgram is a program that will print the full view for the model as the program terminates.
//
// Bubbletea programs limit the output of the view to the terminal size, which fixes issues with rerendering,
// but results in any off-screen output being lost when the program exits.
type FullViewProgram struct {
	*tea.Program
}

func (p *FullViewProgram) Run() (tea.Model, error) {
	model, err := p.Program.Run()

	tea.Batch()

	quittingModel := model.(fullHeightModel)

	quittingModel.quitting = false
	fmt.Println(quittingModel.View())

	return quittingModel.Model, err
}

func NewProgram(model tea.Model, opts ...tea.ProgramOption) *FullViewProgram {
	return &FullViewProgram{tea.NewProgram(fullHeightModel{model, false}, opts...)}
}
