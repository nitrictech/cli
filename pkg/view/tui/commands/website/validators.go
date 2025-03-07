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

	"github.com/nitrictech/cli/pkg/view/tui/components/validation"
)

var (
	nameRegex       = regexp.MustCompile(`^([a-zA-Z0-9-])*$`)
	suffixRegex     = regexp.MustCompile(`[^-]$`)
	prefixRegex     = regexp.MustCompile(`^[^-]`)
	pathPrefixRegex = regexp.MustCompile(`^/`)
	pathRegex       = regexp.MustCompile(`^/[a-zA-Z0-9-]*$`)
)

var websiteNameInFlightValidators = []validation.StringValidator{
	validation.RegexValidator(prefixRegex, "name can't start with a dash"),
	validation.RegexValidator(nameRegex, "name must only contain letters, numbers and dashes"),
}

var websiteNameValidators = append([]validation.StringValidator{
	validation.RegexValidator(suffixRegex, "name can't end with a dash"),
	validation.NotBlankValidator("name can't be blank"),
}, websiteNameInFlightValidators...)

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
