package tui

import (
	"os"

	"github.com/pterm/pterm"
)

func CheckErr(err error) {
	if err != nil {
		pterm.Error.Println(err.Error())
		os.Exit(1)
	}
}
