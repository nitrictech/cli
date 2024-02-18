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
	"errors"
	"regexp"
)

// StringValidator is a function that returns an error if the input is invalid.
type StringValidator func(string) error

func NotBlankValidator(message string) StringValidator {
	return func(value string) error {
		if value == "" {
			return errors.New(message)
		}

		return nil
	}
}

func RegexValidator(regex *regexp.Regexp, message string) StringValidator {
	return func(value string) error {
		if !regex.MatchString(value) {
			return errors.New(message)
		}

		return nil
	}
}

func ComposeValidators(validators ...StringValidator) StringValidator {
	return func(value string) error {
		for _, v := range validators {
			if err := v(value); err != nil {
				return err
			}
		}

		return nil
	}
}

// var alphanumOnly = regexValidator(nameRegex, "name must only contain letters, numbers and dashes")
