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

package aws

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAwsSchedule(t *testing.T) {
	testCases := map[string]struct {
		inputSchedule  string
		wantedSchedule string
		wantedError    error
	}{
		"simple rate": {
			inputSchedule:  "@every 1h30m",
			wantedSchedule: "rate(90 minutes)",
		},
		"missing schedule": {
			inputSchedule: "",
			wantedError:   errors.New(`schedule can not be empty`),
		},
		"one minute rate": {
			inputSchedule:  "@every 1m",
			wantedSchedule: "rate(1 minute)",
		},
		"round to minute if using small units": {
			inputSchedule:  "@every 60000ms",
			wantedSchedule: "rate(1 minute)",
		},
		"malformed rate": {
			inputSchedule: "@every 402 seconds",
			wantedError:   errors.New(`schedule is not valid cron, rate, or preset: failed to parse duration @every 402 seconds: time: unknown unit " seconds" in duration "402 seconds"`),
		},
		"malformed cron": {
			inputSchedule: "every 4m",
			wantedError:   errors.New("schedule is not valid cron, rate, or preset: expected exactly 5 fields, found 2: [every 4m]"),
		},
		"correctly converts predefined schedule": {
			inputSchedule:  "@daily",
			wantedSchedule: "cron(0 0 * * ? *)",
		},
		"unrecognized predefined schedule": {
			inputSchedule: "@minutely",
			wantedError:   errors.New("schedule is not valid cron, rate, or preset: unrecognized descriptor: @minutely"),
		},
		"correctly converts cron with all asterisks": {
			inputSchedule:  "* * * * *",
			wantedSchedule: "cron(* * * * ? *)",
		},
		"correctly converts cron with one ? in DOW": {
			inputSchedule:  "* * * * ?",
			wantedSchedule: "cron(* * * * ? *)",
		},
		"correctly converts cron with one ? in DOM": {
			inputSchedule:  "* * ? * *",
			wantedSchedule: "cron(* * * * ? *)",
		},
		"correctly convert two ? in DOW and DOM": {
			inputSchedule:  "* * ? * ?",
			wantedSchedule: "cron(* * * * ? *)",
		},
		"correctly converts cron with specified DOW": {
			inputSchedule:  "* * * * MON-FRI",
			wantedSchedule: "cron(* * ? * MON-FRI *)",
		},
		"correctly parse provided ? with DOW": {
			inputSchedule:  "* * ? * MON",
			wantedSchedule: "cron(* * ? * MON *)",
		},
		"correctly parse provided ? with DOM": {
			inputSchedule:  "* * 1 * ?",
			wantedSchedule: "cron(* * 1 * ? *)",
		},
		"correctly converts cron with specified DOM": {
			inputSchedule:  "* * 1 * *",
			wantedSchedule: "cron(* * 1 * ? *)",
		},
		"correctly increments 0-indexed DOW": {
			inputSchedule:  "* * ? * 2-6",
			wantedSchedule: "cron(* * ? * 3-7 *)",
		},
		"zero-indexed DOW with un?ed DOM": {
			inputSchedule:  "* * * * 2-6",
			wantedSchedule: "cron(* * ? * 3-7 *)",
		},
		"returns error if both DOM and DOW specified": {
			inputSchedule: "* * 1 * SUN",
			wantedError:   errors.New("parse cron schedule: cannot specify both DOW and DOM in cron expression"),
		},
		"returns error if fixed interval less than one minute": {
			inputSchedule: "@every -5m",
			wantedError:   errors.New("parse fixed interval: duration must be greater than or equal to 1 minute"),
		},
		"returns error if fixed interval is 0": {
			inputSchedule: "@every 0m",
			wantedError:   errors.New("parse fixed interval: duration must be greater than or equal to 1 minute"),
		},
		"error on non-whole-number of minutes": {
			inputSchedule: "@every 89s",
			wantedError:   errors.New("parse fixed interval: duration must be a whole number of minutes or hours"),
		},
		"error on too many inputs": {
			inputSchedule: "* * * * * *",
			wantedError:   errors.New(`schedule is not valid cron, rate, or preset: expected exactly 5 fields, found 6: [* * * * * *]`),
		},
		"cron syntax error": {
			inputSchedule: "* * * malformed *",
			wantedError:   errors.New(`schedule is not valid cron, rate, or preset: failed to parse int from malformed: strconv.Atoi: parsing "malformed": invalid syntax`),
		},
		"passthrogh AWS flavored cron": {
			inputSchedule:  "cron(0 * * * ? *)",
			wantedSchedule: "cron(0 * * * ? *)",
		},
		"passthrough AWS flavored rate": {
			inputSchedule:  "rate(5 minutes)",
			wantedSchedule: "rate(5 minutes)",
		},
		"Given an expression with more than 5 values": {
			inputSchedule: "*/1 * * * ? *",
			wantedError:   errors.New(`schedule is not valid cron, rate, or preset: expected exactly 5 fields, found 6: [*/1 * * * ? *]`),
		},
		"Given a valid cron expression": {
			inputSchedule:  "*/1 * * * *",
			wantedSchedule: "cron(0/1 * * * ? *)",
		},
		"Given a valid cron with a Day of Week value between 0-6": {
			inputSchedule:  "*/1 * * * 3",
			wantedSchedule: "cron(0/1 * ? * 4 *)",
		},
		"Given a valid cron with a Day of Week value of 0 (Sunday)": {
			inputSchedule:  "*/1 * * * 0",
			wantedSchedule: "cron(0/1 * ? * 1 *)",
		},
		"Given a valid cron with a Day of Week value range": {
			inputSchedule:  "*/1 * * * 1-3",
			wantedSchedule: "cron(0/1 * ? * 2-4 *)",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			parsedSchedule, err := awsSchedule(tc.inputSchedule)

			if tc.wantedError != nil {
				require.EqualError(t, err, tc.wantedError.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantedSchedule, parsedSchedule)
			}
		})
	}
}
