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
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func GoModule(searchPath string) (string, error) {
	files, err := FindFilesInDir(searchPath, "go.mod")
	if err != nil {
		return "", err
	}

	for _, fname := range files {
		f, err := os.Open(filepath.Join(searchPath, fname))
		if err != nil {
			return "", err
		}

		defer f.Close()

		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			cols := strings.Split(scanner.Text(), " ")
			return cols[1], nil
		}
	}

	return "", errors.New("no valid go.mod found")
}
