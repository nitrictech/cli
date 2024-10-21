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

package validation

import (
	"fmt"
	"regexp"

	"github.com/ettle/strcase"
)

var ResourceName_Rule = &Rule{
	name: "Invalid Name",
	// TODO: Add docs link for rule when available
	docsUrl: "",
}

var lowerKebabCaseRe, _ = regexp.Compile("^[a-z0-9]+(-[a-z0-9]+)*$")

func IsValidResourceName(name string) bool {
	return lowerKebabCaseRe.Match([]byte(name))
}

func NewResourceNameViolationError(resourceName string, resourceType string) *RuleViolationError {
	return ResourceName_Rule.newError(fmt.Sprintf("'%s' for %s try '%s'", resourceName, resourceType, strcase.ToKebab(resourceName)))
}
