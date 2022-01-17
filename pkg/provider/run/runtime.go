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
package run

import (
	"fmt"
	"path/filepath"

	"github.com/docker/docker/api/types/strslice"
	"github.com/spf13/cobra"

	"github.com/nitrictech/newcli/pkg/build"
)

type Runtime string

const (
	RuntimeTypescript Runtime = "ts"
	RuntimeJavascript Runtime = "js"
)

func devImageNameForRuntime(runtime Runtime) string {
	return fmt.Sprintf("nitric-%s-dev", runtime)
}

type LaunchOpts struct {
	Entrypoint []string
	Cmd        []string
}

func launchOptsForFunction(f *Function) (LaunchOpts, error) {
	switch f.runtime {
	case RuntimeJavascript:
		// Javascript will re-use typescript runtime
		fallthrough
	case RuntimeTypescript:
		return LaunchOpts{
			Entrypoint: strslice.StrSlice{"nodemon"},
			Cmd:        strslice.StrSlice{"--watch", "/app/**", "--ext", "ts,js,json", "--exec", "ts-node -T " + "/app/" + f.handler},
		}, nil
	}

	return LaunchOpts{}, fmt.Errorf("unsupported runtime")
}

func CreateBaseDevForFunctions(funcs []*Function) error {
	ctx, _ := filepath.Abs(".")
	if err := build.CreateBaseDev(ctx, map[string]string{
		"ts": "nitric-ts-dev",
	}); err != nil {
		cobra.CheckErr(err)
	}

	imageBuilds := make(map[string]string)

	for _, f := range funcs {
		switch f.runtime {
		case RuntimeJavascript:
			// Javascript will re-use typescript runtime
			fallthrough
		case RuntimeTypescript:
			imageBuilds[string(RuntimeTypescript)] = devImageNameForRuntime(RuntimeTypescript)
		}
	}

	// Currently the file context does not matter (base runtime images should not copy files)
	return build.CreateBaseDev(".", imageBuilds)
}
