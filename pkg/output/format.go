package output

import (
	"encoding/json"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/newcli/pkg/pflagext"
)

var (
	allowedFormats = []string{"json", "yaml", "table"}
	defaultFormat  = "table"
	outputFormat   string
	OutputTypeFlag = pflagext.NewStringEnumVar(&outputFormat, allowedFormats, defaultFormat)
)

func Print(object interface{}) {
	switch outputFormat {
	case "json":
		printJson(object)
	case "yaml":
		printYaml(object)
	default:
		printTable(object)
	}
}

func printJson(object interface{}) {
	b, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func printYaml(object interface{}) {
	b, err := yaml.Marshal(object)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func printTable(object interface{}) {
	// TODO research a good printer
	spew.Dump(object)
}
