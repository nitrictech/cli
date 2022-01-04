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
	"io/fs"
	"time"
)

type stringFileInfo struct {
	name string
	size int64
}

func NewStringFileInfo(fileName, content string) fs.FileInfo {
	return &stringFileInfo{name: fileName, size: int64(len(content))}
}

func (i *stringFileInfo) Name() string       { return i.name }
func (i *stringFileInfo) Size() int64        { return i.size }
func (i *stringFileInfo) Mode() fs.FileMode  { return 0 }
func (i *stringFileInfo) ModTime() time.Time { return time.Now() }
func (i *stringFileInfo) IsDir() bool        { return false }
func (i *stringFileInfo) Sys() interface{}   { return nil }
