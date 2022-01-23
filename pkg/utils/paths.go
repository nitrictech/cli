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
	"log"
	"os"
	"os/user"
	"path"
	"runtime"
	"strings"
)

// slashSplitter - used to split strings, with the same output regardless of leading or trailing slashes
// e.g - strings.FieldsFunc("/one/two/three/", f) == strings.FieldsFunc("/one/two/three", f) == strings.FieldsFunc("one/two/three", f) == ["one" "two" "three"]
func slashSplitter(c rune) bool {
	return c == '/'
}

// SplitPath - splits a path into its component parts, ignoring leading or trailing slashes.
// e.g - SplitPath("/one/two/three/") == SplitPath("/one/two/three") == SplitPath("one/two/three") == ["one" "two" "three"]
func SplitPath(p string) []string {
	return strings.FieldsFunc(p, slashSplitter)
}

// homeDir gets the nitric home directory
func homeDir() string {
	nitricHomeEnv := os.Getenv("NITRIC_HOME")
	if nitricHomeEnv != "" {
		return nitricHomeEnv
	}

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	return path.Join(dirname, ".nitric")
}

// NitricRunDir returns the directory to place runtime data.
func NitricRunDir() string {
	if runtime.GOOS == "linux" {
		u, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		return path.Join("/run/user/", u.Uid, "nitric")
	}
	return path.Join(homeDir(), "run")
}

// NitricTemplatesDir returns the directory to place template related data.
func NitricTemplatesDir() string {
	return path.Join(homeDir(), "store")
}

// NitricConfigDir returns the directory to find configuration.
func NitricConfigDir() string {
	if runtime.GOOS == "linux" {
		dirname, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}

		return path.Join(dirname, ".config", "nitric")
	}
	return homeDir()
}

// NitricLogDir returns the directory to find log files.
func NitricLogDir(stackPath string) string {
	return path.Join(stackPath, ".nitric")
}
