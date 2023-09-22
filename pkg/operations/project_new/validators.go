package project_new

import (
	"errors"
	"regexp"
	"strings"
)

var nameRegex = regexp.MustCompile(`^([a-zA-Z0-9-])*$`)

// validateName validates whether the input string is a valid project name.
//
// inFlight: indicates whether to use the inflight validation (loose) or not (strict)
//   - true: inFlight mode, name endings are not validated since the text is assumed incomplete.
//   - false: strict mode, all characters are validated.
func validateName(projectName string, inFlight bool) error {
	if projectName == "" {
		return errors.New("name can't be empty")
	}

	if strings.HasPrefix(projectName, "-") {
		return errors.New("name can't start with a dash")
	}

	if !inFlight && strings.HasSuffix(projectName, "-") {
		return errors.New("name can't end with a dash")
	}

	if !nameRegex.MatchString(projectName) {
		return errors.New("name must only contain letters, numbers and dashes")
	}

	return nil
}
