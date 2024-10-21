// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
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
	name    string
	docsUrl string
}

func (r *Rule) newError(message string) *RuleViolationError {
	return &RuleViolationError{
		rule:    r,
		message: message,
	}
}

func (r *Rule) String() string {
	return fmt.Sprintf("%s: %s", r.name, r.docsUrl)
}

type RuleViolationError struct {
	rule    *Rule
	message string
}

func (r *RuleViolationError) Error() string {
	return fmt.Sprintf("%s: %s", r.rule.name, r.message)
}

func GetRuleViolation(err error) *Rule {
	ruleViolation := &RuleViolationError{}

	if errors.As(err, &ruleViolation) {
		return ruleViolation.rule
	}

	return nil
}
