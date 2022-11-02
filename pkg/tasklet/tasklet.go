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

package tasklet

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/pterm/pterm"

	"github.com/nitrictech/cli/pkg/output"
)

var defaultSequence = []string{"⠟", "⠯", "⠷", "⠾", "⠽", "⠻"}

type TaskletFn func(output.Progress) error

type Runner struct {
	Runner   TaskletFn
	StartMsg string
	StopMsg  string
}

type Opts struct {
	Signal        chan os.Signal
	Timeout       time.Duration
	SuccessPrefix string
}

type taskletContext struct {
	spinner *pterm.SpinnerPrinter
}

var _ output.Progress = &taskletContext{}

func (c *taskletContext) Debugf(format string, a ...interface{}) {
	if output.CI {
		if output.VerboseLevel > 1 {
			fmt.Println(strings.TrimFunc(fmt.Sprintf(format, a...), unicode.IsSpace))
		}
	} else {
		pterm.Debug.Println(fmt.Sprintf(format, a...))
	}
}

func (c *taskletContext) Busyf(format string, a ...interface{}) {
	if !output.CI {
		c.spinner.UpdateText(fmt.Sprintf(format, a...))
	}
}

func (c *taskletContext) Successf(format string, a ...interface{}) {
	if output.CI {
		fmt.Println(strings.TrimFunc(fmt.Sprintf(format, a...), unicode.IsSpace))
	} else {
		c.spinner.SuccessPrinter.Printf(format, a...)
	}
}

func (c *taskletContext) Failf(format string, a ...interface{}) {
	if output.CI {
		fmt.Println(strings.TrimFunc(fmt.Sprintf(format, a...), unicode.IsSpace))
	} else {
		pterm.Error.Printf(format, a...)
	}
}

func MustRun(runner Runner, opts Opts) {
	if Run(runner, opts) != nil {
		os.Exit(1)
	}
}

func Run(runner Runner, opts Opts) error {
	spinner, err := pterm.DefaultSpinner.WithShowTimer().WithSequence(defaultSequence...).Start(runner.StartMsg)
	if err != nil {
		return err
	}

	defer func() {
		_ = spinner.Stop()
	}()

	if opts.SuccessPrefix != "" {
		spinner.SuccessPrinter = &pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.SuccessMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.SuccessPrefixStyle,
				Text:  opts.SuccessPrefix,
			},
		}
	}

	tCtx := &taskletContext{spinner: spinner}

	start := time.Now()
	done := make(chan bool, 1)
	doErr := make(chan error, 1)

	if opts.Timeout == 0 {
		opts.Timeout = time.Hour // our infinite
	}

	timer := time.NewTimer(opts.Timeout)

	go func() {
		err = runner.Runner(tCtx)
		if err != nil {
			doErr <- err
		}

		done <- true
	}()

	select {
	case err = <-doErr:
	case <-timer.C:
		err = errors.New("tasklet timedout after " + opts.Timeout.String())
	case <-done:
	case <-opts.Signal:
		fmt.Println("Shutting down services - exiting")
	}

	elapsed := time.Since(start)
	if elapsed < time.Second {
		time.Sleep(time.Second - elapsed)
	}

	if err != nil {
		spinner.Fail(err)
		return err
	}

	spinner.SuccessPrinter.Printf("%s (%s)", runner.StopMsg, elapsed.Round(time.Second).String())

	return nil
}
