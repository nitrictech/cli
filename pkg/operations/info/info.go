package info

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nitrictech/cli/pkg/ghissue"
	"github.com/nitrictech/cli/pkg/output"
)

func Run(ctx context.Context) {
	d := ghissue.Gather()

	s, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		output.Print(d)
	} else {
		fmt.Println(string(s))
	}
}
