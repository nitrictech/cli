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
	"fmt"
	"os"

	"github.com/pterm/pterm"
)

func CheckErr(err error) {
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
}

func NewIncompatibleWorkerError() error {
	return fmt.Errorf("unable to register incompatible worker. This can be caused by out of date Nitric CLI versions, an upgrade may resolve this issue")
}
