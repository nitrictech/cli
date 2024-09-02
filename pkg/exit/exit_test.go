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

package exit

import (
	"errors"
	"testing"
)

func TestExitService_GetExitService(t *testing.T) {
	service1 := GetExitService()
	service2 := GetExitService()

	if service1 != service2 {
		t.Errorf("Expected GetExitService() to return the same instance, but got different instances")
	}
}

func TestExitService_Exit(t *testing.T) {
	service := GetExitService()

	// Track if the exit function is called
	exitCalled := false

	// Subscribe to exit event
	service.SubscribeToExit(func(err error) {
		exitCalled = true
	})

	// Test case 1: Exit with nil error
	service.Exit(nil)

	if !exitCalled {
		t.Errorf("Expected exit to be called with nil error")
	}

	// Reset and test case 2: Exit with non-nil error
	service.SubscribeToExit(func(err error) {
		exitCalled = true
	})

	exitCalled = false
	err := errors.New("something went wrong")
	service.Exit(err)

	if !exitCalled {
		t.Errorf("Expected exit to be called with non-nil error")
	}
}

func TestExitService_SubscribeToExit(t *testing.T) {
	service := GetExitService()

	// Track if the subscription function is called
	subscriptionCalled := false
	subscription := func(err error) {
		subscriptionCalled = true
	}

	// Test case 1: Subscribe to exit event
	service.SubscribeToExit(subscription)
	service.Exit(nil)

	if !subscriptionCalled {
		t.Errorf("Expected subscription to be called on exit")
	}

	// Test case 2: Ensure that subsequent SubscribeToExit calls do not change the behavior
	subscriptionCalled = false

	service.Exit(nil)

	if subscriptionCalled {
		t.Errorf("Expected the first subscription to not be called on exit, but it was")
	}
}
