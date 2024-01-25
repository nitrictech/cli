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

package stack

import (
	"regexp"

	"github.com/nitrictech/cli/pkgplus/view/tui/components/validation"
)

var (
	nameRegex         = regexp.MustCompile(`^([a-zA-Z0-9-])*$`)
	suffixRegex       = regexp.MustCompile(`[^-]$`)
	prefixRegex       = regexp.MustCompile(`^[^-]`)
	emailRegex        = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	azureOrgNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,48}[a-zA-Z0-9]$`)
	gcpProjectIDRegex = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
)

var ProjectNameInFlightValidators = []validation.StringValidator{
	validation.RegexValidator(prefixRegex, "name can't start with a dash"),
	validation.RegexValidator(nameRegex, "name must only contain letters, numbers and dashes"),
}

var ProjectNameValidators = append([]validation.StringValidator{
	validation.RegexValidator(suffixRegex, "name can't end with a dash"),
	validation.NotBlankValidator("name can't be blank"),
}, ProjectNameInFlightValidators...)
