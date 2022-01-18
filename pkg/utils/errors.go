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

package utils

import (
	"errors"
	"strings"
	"sync"
)

func NewErrorList() *ErrorList {
	return &ErrorList{
		lock: &sync.RWMutex{},
	}
}

// ErrorList is for when one error is not the cause of others.
type ErrorList struct {
	lock sync.Locker
	errs []error
}

func (e *ErrorList) Add(err error) {
	e.lock.Lock()
	defer e.lock.Unlock()
	if err == nil {
		return
	}
	e.errs = append(e.errs, err)
}

func (e *ErrorList) Error() string {
	e.lock.Lock()
	defer e.lock.Unlock()
	msgs := []string{}
	for _, m := range e.errs {
		msgs = append(msgs, m.Error())
	}
	return strings.Join(msgs, "\n")
}

func (e *ErrorList) Aggregate() error {
	if len(e.errs) == 0 {
		return nil
	}
	return e
}

// NotSupportedError indicates that a request operation cannot be performed,
// because it is unsupported.
// Functions and methods should not return this error but should instead
// return an error including appropriate context that satisfies
//     errors.Is(err, errors.NotSupportedError)
// either by directly wrapping NotSupportedError or by implementing an Is method.
type NotSupportedError struct {
	error
}

func NewNotSupportedErr(message string) error {
	return &NotSupportedError{error: errors.New(message)}
}

func (*NotSupportedError) Is(err error) bool {
	return strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "not supported")
}
