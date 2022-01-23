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

package pflagext

import (
	"errors"
	"os"
	"path/filepath"
)

type AllowOpt int

const (
	AllowFileAndDir AllowOpt = 0
	AllowFileOnly   AllowOpt = 1
	AllowDirOnly    AllowOpt = 2
)

type pathFlag struct {
	Allow  AllowOpt
	ValueP *string
}

// NewPathVar creates a path flag that will validate the path given the allowOpt.
func NewPathVar(value *string, allow AllowOpt, def string) *pathFlag {
	*value = def
	return &pathFlag{
		Allow:  allow,
		ValueP: value,
	}
}

func (p *pathFlag) String() string {
	return *p.ValueP
}

// Set will validate the path, and make sure it exists and correctly a dir/file or either.
func (p *pathFlag) Set(n string) error {
	nPath, err := filepath.Abs(n)
	if err != nil {
		return err
	}

	ss, err := os.Stat(nPath)
	if err != nil {
		return err
	}

	if p.Allow == AllowDirOnly && !ss.IsDir() {
		return errors.New("provided path is a file, expected a directory")
	}

	if p.Allow == AllowFileOnly && ss.IsDir() {
		return errors.New("provided path is a directory, expected a file")
	}

	*p.ValueP = nPath
	return nil
}

func (p *pathFlag) Type() string {
	return "pathVar"
}
