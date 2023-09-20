package logger

import (
	"os"

	"github.com/charmbracelet/log"
)

// Simple logger for printing out to the command line
var Simple = log.NewWithOptions(
	os.Stdout,
	log.Options{
		ReportTimestamp: false,
	},
)
