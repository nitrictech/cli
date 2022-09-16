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

package pulumi

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/debug"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"

	"github.com/nitrictech/cli/pkg/output"
)

func updateLoggingOpts(log *pulumiLogger) []optup.Option {
	upChannel := make(chan events.EngineEvent)
	opts := []optup.Option{
		optup.EventStreams(upChannel),
	}

	go collectEvents(log, upChannel, "Deploying.. ")

	if output.VerboseLevel >= 2 {
		piper, pipew := io.Pipe()

		go output.StdoutToPtermDebug(piper, log, "Deploying.. ")

		opts = append(opts, optup.ProgressStreams(pipew))
	}

	if output.VerboseLevel > 2 {
		var loglevel uint = uint(output.VerboseLevel)

		opts = append(opts, optup.DebugLogging(debug.LoggingOptions{
			LogLevel:      &loglevel,
			LogToStdErr:   true,
			FlowToPlugins: true,
		}))
	}

	return opts
}

func destroyLoggingOpts(log *pulumiLogger) []optdestroy.Option {
	upChannel := make(chan events.EngineEvent)
	opts := []optdestroy.Option{
		optdestroy.EventStreams(upChannel),
	}

	go collectEvents(log, upChannel, "Deleting.. ")

	if output.VerboseLevel >= 2 {
		piper, pipew := io.Pipe()
		go output.StdoutToPtermDebug(piper, log, "Deleting.. ")

		opts = append(opts, optdestroy.ProgressStreams(pipew))
	}

	if output.VerboseLevel > 2 {
		var loglevel uint = uint(output.VerboseLevel)

		opts = append(opts, optdestroy.DebugLogging(debug.LoggingOptions{
			LogLevel:      &loglevel,
			LogToStdErr:   true,
			FlowToPlugins: true,
		}))
	}

	return opts
}

func stepEventToString(eType string, evt *apitype.StepEventMetadata) string {
	urnSplit := strings.Split(evt.URN, "::")
	name := urnSplit[len(urnSplit)-1]

	typeSplit := strings.Split(evt.Type, ":")
	rType := typeSplit[len(typeSplit)-1]

	return fmt.Sprintf("%s/%s", rType, name)
}

const busyMsg = "%s %d/%d resources (%d failed)"

func collectEvents(log *pulumiLogger, eventChannel <-chan events.EngineEvent, prefix string) {
	busyList := map[string]time.Time{}

	busy := 0
	done := 0
	failed := 0

	for {
		var (
			event events.EngineEvent
			ok    bool
		)

		event, ok = <-eventChannel
		if !ok {
			return
		}

		log.CollectEvent(event)

		if event.ResourcePreEvent != nil && event.ResourcePreEvent.Metadata.Op != apitype.OpSame {
			busy++

			lastCreating := stepEventToString("ResourcePreEvent", &event.ResourcePreEvent.Metadata)

			busyList[lastCreating] = time.Now()

			log.Busyf(busyMsg, prefix, done, busy, failed)
		}

		if event.ResOutputsEvent != nil {
			lc := stepEventToString("ResOutputsEvent", &event.ResOutputsEvent.Metadata)

			if event.ResOutputsEvent.Metadata.Op == apitype.OpSame {
				log.Debugf("%s", lc)
			} else {
				if st, ok := busyList[lc]; ok {
					// if possible print out how long it took
					d := time.Since(st).Round(time.Second)
					log.Successf("%s (%s)", lc, d.String())
				} else {
					log.Successf("%s %t", lc, busyList[lc])
				}
			}

			done++
			log.Busyf(busyMsg, prefix, done, busy, failed)
		}

		if event.ResOpFailedEvent != nil {
			lc := stepEventToString("ResOpFailedEvent", &event.ResOpFailedEvent.Metadata)

			log.Failf("%s", lc)

			delete(busyList, lc)

			done++
			failed++

			if len(busyList) > 0 {
				log.Busyf(busyMsg, prefix, done, busy, failed)
			}
		}
	}
}
