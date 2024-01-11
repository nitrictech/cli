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

package cron

import (
	"fmt"
	"strconv"
	"strings"
)

// RateToCron - Converts a valid rate expression
// into a simple crontab expression
func RateToCron(rate string) (string, error) {
	rateParts := strings.Split(rate, " ")
	if len(rateParts) < 2 {
		return "", fmt.Errorf("not enough parts to rate expression %s", rate)
	}

	rateNum := rateParts[0]
	rateType := rateParts[1]

	num, err := strconv.Atoi(rateNum)
	if err != nil {
		return "", fmt.Errorf("invalid rate expression %s; %w", rate, err)
	}

	switch rateType {
	case "minutes":
		// Every nth minute
		return fmt.Sprintf("*/%d * * * *", num), nil
	case "hours":
		// The top of every nth hour
		return fmt.Sprintf("0 */%d * * *", num), nil
	case "days":
		// Midnight every nth day
		return fmt.Sprintf("0 0 */%d * *", num), nil
	default:
		return "", fmt.Errorf("invalid rate expression %s; %s must be one of [minutes, hours, days]", rate, rateType)
	}
}
