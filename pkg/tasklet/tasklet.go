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
	"time"

	"github.com/pterm/pterm"
)

var defaultSequence = []string{"⠟", "⠯", "⠷", "⠾", "⠽", "⠻"}

type TaskletFn func(TaskletContext) error

type Runner struct {
	Runner   TaskletFn
	StartMsg string
	StopMsg  string
}

type Opts struct {
	Signal  chan os.Signal
	Timeout time.Duration
}

type TaskletContext interface {
	Spinner() *pterm.SpinnerPrinter
}

type taskletContext struct {
	spinner *pterm.SpinnerPrinter
}

func (c *taskletContext) Spinner() *pterm.SpinnerPrinter {
	return c.spinner
}

func MustRun(runner Runner, opts Opts) {
	if Run(runner, opts) != nil {
		os.Exit(1)
	}
}

func Run(runner Runner, opts Opts) error {
	spinner, err := pterm.DefaultSpinner.WithSequence(defaultSequence...).Start(runner.StartMsg)
	if err != nil {
		return err
	}

	start := time.Now()
	done := make(chan bool, 1)
	doErr := make(chan error, 1)

	if opts.Timeout == 0 {
		opts.Timeout = time.Hour // our infinite
	}
	timer := time.NewTimer(opts.Timeout)

	go func() {
		err = runner.Runner(&taskletContext{spinner: spinner})
		if err != nil {
			doErr <- err
		}
		done <- true
	}()
	select {
	case err = <-doErr:
	case <-timer.C:
		err = errors.New("tasklet timedout after " + time.Since(start).String())
	case <-done:
	case sigTerm := <-opts.Signal:
		err = fmt.Errorf("received %v, exiting", sigTerm)
	}

	elapsed := time.Since(start)
	if elapsed < time.Second {
		time.Sleep(time.Second - elapsed)
	}

	if err != nil {
		spinner.Fail(err)
		return err
	}

	spinner.Success(runner.StopMsg)
	return nil
}
