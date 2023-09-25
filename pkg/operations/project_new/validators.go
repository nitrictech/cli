package project_new

import (
	"regexp"

	"github.com/nitrictech/cli/pkg/tui/validation"
)

var nameRegex = regexp.MustCompile(`^([a-zA-Z0-9-])*$`)
var suffixRegex = regexp.MustCompile(`[^-]$`)
var prefixRegex = regexp.MustCompile(`^[^-]`)

var projectNameInFlightValidators = []validation.StringValidator{
	validation.RegexValidator(prefixRegex, "name can't start with a dash"),
	validation.RegexValidator(nameRegex, "name must only contain letters, numbers and dashes"),
}

var projectNameValidators = append([]validation.StringValidator{
	validation.RegexValidator(suffixRegex, "name can't end with a dash"),
	validation.NotBlankValidator("name can't be blank"),
}, projectNameInFlightValidators...)
