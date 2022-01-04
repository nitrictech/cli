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
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/jhoonb/archivex"
)

func TarReaderFromString(name, src string) (io.Reader, error) {
	tar := new(archivex.TarFile)
	tarReader := bytes.Buffer{}
	err := tar.CreateWriter(name+".tar", &tarReader)
	if err != nil {
		return nil, err
	}

	err = tar.Add(src, strings.NewReader(src), NewStringFileInfo(name, src))
	if err != nil {
		return nil, err
	}

	tar.Close()

	return &tarReader, nil
}

func TarReaderFromPath(src string) (io.Reader, error) {
	tar := new(archivex.TarFile)
	tarReader := bytes.Buffer{}
	err := tar.CreateWriter(src+".tar", &tarReader)
	if err != nil {
		return nil, err
	}

	ss, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	if ss.IsDir() {
		err = tar.AddAll(src, false)
		if err != nil {
			return nil, err
		}
	} else {
		file, err := os.Open(src)
		if err != nil {
			return nil, err
		}

		err = tar.Add(src, file, ss)
		if err != nil {
			return nil, err
		}
	}

	tar.Close()

	return &tarReader, nil
}
