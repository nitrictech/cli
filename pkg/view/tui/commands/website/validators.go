// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package add_website

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/nitrictech/cli/pkg/view/tui/components/validation"
)

var (
	pathPrefixRegex = regexp.MustCompile(`^/`)
	pathRegex       = regexp.MustCompile(`^/[a-zA-Z0-9-]*$`)
	// WebsiteNameRegex matches valid characters for a website name
	WebsiteNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	// WebsiteNameStartRegex ensures the name starts with a letter or number
	WebsiteNameStartRegex = regexp.MustCompile(`^[a-zA-Z0-9]`)
	// WebsiteNameEndRegex ensures the name doesn't end with a hyphen
	WebsiteNameEndRegex = regexp.MustCompile(`[a-zA-Z0-9]$`)
)

func WebsiteNameInFlightValidators() []validation.StringValidator {
	return []validation.StringValidator{
		validation.NotBlankValidator("Website name is required"),
		validation.RegexValidator(WebsiteNameRegex, "Website name can only contain letters, numbers, underscores and hyphens"),
		validation.RegexValidator(WebsiteNameStartRegex, "Website name must start with a letter or number"),
		validation.RegexValidator(WebsiteNameEndRegex, "Website name cannot end with a hyphen"),
	}
}

func WebsiteNameValidators(existingNames []string) []validation.StringValidator {
	return append([]validation.StringValidator{
		validation.NotBlankValidator("Website name is required"),
		validation.RegexValidator(WebsiteNameRegex, "Website name can only contain letters, numbers, underscores and hyphens"),
		validation.RegexValidator(WebsiteNameStartRegex, "Website name must start with a letter or number"),
		validation.RegexValidator(WebsiteNameEndRegex, "Website name cannot end with a hyphen"),
	}, func(existingNames []string) validation.StringValidator {
		return func(value string) error {
			// Normalize the new value
			newName := strings.TrimPrefix(value, "./")

			for _, name := range existingNames {
				// Normalize the existing basedir
				existingName := strings.TrimPrefix(name, "./")
				if existingName == newName {
					return fmt.Errorf("website name already exists")
				}
			}

			return nil
		}
	}(existingNames))
}

func DisallowedPathsValidator(disallowedPaths []string) validation.StringValidator {
	return func(value string) error {
		if slices.Contains(disallowedPaths, value) {
			return fmt.Errorf("duplicate path '%s' is not allowed", value)
		}

		return nil
	}
}

func WebsiteURLPathInFlightValidators(disallowedPaths []string) []validation.StringValidator {
	return []validation.StringValidator{
		validation.RegexValidator(pathPrefixRegex, "path must start with a slash"),
		validation.RegexValidator(pathRegex, "path must only contain letters, numbers, and dashes after the initial slash"),
		DisallowedPathsValidator(disallowedPaths),
	}
}

func WebsiteURLPathValidators(disallowedPaths []string) []validation.StringValidator {
	return append([]validation.StringValidator{
		validation.NotBlankValidator("path can't be blank"),
	}, WebsiteURLPathInFlightValidators(disallowedPaths)...)
}

// PortValidators returns a list of validators for the port field
func PortValidators() []validation.StringValidator {
	return []validation.StringValidator{
		func(value string) error {
			if value == "" {
				return fmt.Errorf("port cannot be empty")
			}

			port, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("port must be a number")
			}

			if port < 1 || port > 65535 {
				return fmt.Errorf("port must be between 1 and 65535")
			}

			return nil
		},
	}
}

// PortInFlightValidators returns a list of in-flight validators for the port field
func PortInFlightValidators() []validation.StringValidator {
	return []validation.StringValidator{
		func(value string) error {
			if value == "" {
				return nil // Allow empty during typing
			}

			// Check if it's a number
			if _, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("port must be a number")
			}

			return nil
		},
	}
}
