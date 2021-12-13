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
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	stackPath string
)

func wrapStatError(err error) error {
	if os.IsNotExist(err) {
		return errors.WithMessage(err, "Please provide the correct path to the stack (eg. -s ./nitric.yaml)")
	}
	if os.IsPermission(err) {
		return errors.WithMessagef(err, "Please make sure that %s has the correct permissions", stackPath)
	}
	return err
}

func FromOptions() (*Stack, error) {
	ss, err := os.Stat(stackPath)
	if err != nil {
		return nil, wrapStatError(err)
	}
	if ss.IsDir() {
		stackPath = path.Join(stackPath, "nitric.yaml")
	}
	_, err = os.Stat(stackPath)
	if err != nil {
		return nil, wrapStatError(err)
	}

	return FromFile(stackPath)
}

func AddOptions(cmd *cobra.Command) {
	wd, err := os.Getwd()
	cobra.CheckErr(err)
	cmd.Flags().StringVarP(&stackPath, "stack", "s", wd, "path to the nitric.yaml stack")
}
