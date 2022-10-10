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

package output

import (
	"bufio"
	"io"
	"strings"
	"unicode"

	"github.com/pterm/pterm"
)

var (
	VerboseLevel int
	CI           bool
)

type Progress interface {
	Debugf(format string, a ...interface{})
	Busyf(format string, a ...interface{})
	Successf(format string, a ...interface{})
	Failf(format string, a ...interface{})
}

func StdoutToPtermDebug(b io.ReadCloser, p Progress, prefix string) {
	defer b.Close()

	sc := bufio.NewScanner(b)
	for sc.Scan() {
		line := strings.TrimRightFunc(sc.Text(), unicode.IsSpace)

		if line != "" {
			p.Debugf("%s %v", prefix, line)
		}
	}
}

type pTermWriter struct {
	prefix pterm.PrefixPrinter
}

// Write is here to so that pTermWriter can be used as an io.Writer
// eg. `command.stdout = NewPtermWriter()`
func (p *pTermWriter) Write(b []byte) (n int, err error) {
	p.prefix.Print(string(b))

	return len(b), nil
}

func NewPtermWriter(prefix pterm.PrefixPrinter) *pTermWriter {
	return &pTermWriter{prefix: prefix}
}
