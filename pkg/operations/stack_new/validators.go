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

package stack_new

import (
	"regexp"

	"github.com/nitrictech/pearls/pkg/tui/validation"
)

var (
	nameRegex         = regexp.MustCompile(`^([a-zA-Z0-9-])*$`)
	suffixRegex       = regexp.MustCompile(`[^-]$`)
	prefixRegex       = regexp.MustCompile(`^[^-]`)
	emailRegex        = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	azureOrgNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,48}[a-zA-Z0-9]$`)
	gcpProjectIDRegex = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
)

var projectNameInFlightValidators = []validation.StringValidator{
	validation.RegexValidator(prefixRegex, "name can't start with a dash"),
	validation.RegexValidator(nameRegex, "name must only contain letters, numbers and dashes"),
}

var projectNameValidators = append([]validation.StringValidator{
	validation.RegexValidator(suffixRegex, "name can't end with a dash"),
	validation.NotBlankValidator("name can't be blank"),
}, projectNameInFlightValidators...)

var azureOrgNameValidators = []validation.StringValidator{validation.RegexValidator(azureOrgNameRegex, "org must start and end with a letter/number, can include hyphens, and be under 50 characters.")}

var adminEmailValidators = []validation.StringValidator{validation.RegexValidator(emailRegex, "admin email must be a valid email address")}

var gcpProjectIDValidators = []validation.StringValidator{validation.RegexValidator(gcpProjectIDRegex, "GCP project ID must be 6 to 30 lowercase letters, digits, or hyphens")}
