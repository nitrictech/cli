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
	"strings"

	"github.com/nitrictech/cli/pkg/view/tui/components/validation"
)

var (
	pathPrefixRegex = regexp.MustCompile(`^/`)
	pathRegex       = regexp.MustCompile(`^/[a-zA-Z0-9-]*$`)
)

func WebsiteNameInFlightValidators() []validation.StringValidator {
	return []validation.StringValidator{
		validation.NotBlankValidator("Website name is required"),
		validation.RegexValidator(regexp.MustCompile(`^[a-zA-Z0-9_-]*$`), "Website name can only contain letters, numbers, underscores and hyphens"),
	}
}

func WebsiteNameValidators(existingNames []string) []validation.StringValidator {
	return append([]validation.StringValidator{
		validation.NotBlankValidator("Website name is required"),
		validation.RegexValidator(regexp.MustCompile(`^[a-zA-Z0-9_-]+$`), "Website name can only contain letters, numbers, underscores and hyphens"),
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
