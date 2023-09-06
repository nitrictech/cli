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

package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/docker/distribution/reference"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	goruntime "runtime"

	"github.com/nitrictech/cli/pkg/containerengine"
	"github.com/nitrictech/cli/pkg/project"
)

func dynamicDockerfile(dir, name string) (*os.File, error) {
	// create a more stable file name for the hashing
	return os.Create(filepath.Join(dir, fmt.Sprintf("%s.nitric.dynamic.dockerfile", name)))
}

func buildFunction(s *project.Project, f project.Function) func() error {
	fun := &f

	return func() error {
		ce, err := containerengine.Discover()
		if err != nil {
			return err
		}

		rt, err := fun.GetRuntime()
		if err != nil {
			return err
		}

		f, err := dynamicDockerfile(s.Dir, fun.Name)
		if err != nil {
			return err
		}

		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()

		if err := rt.BaseDockerFile(f); err != nil {
			return err
		}

		ingoreFunctions := lo.Filter(lo.Values(s.Functions), func(item project.Function, index int) bool {
			return item.Name != fun.Name
		})

		ignoreHandlers := lo.Map(ingoreFunctions, func(item project.Function, index int) string {
			return item.Handler
		})

		ignores := rt.BuildIgnore(ignoreHandlers...)

		if err := ce.Build(filepath.Base(f.Name()), s.Dir, fmt.Sprintf("%s-%s", s.Name, fun.Name), rt.BuildArgs(), ignores); err != nil {
			return err
		}

		return nil
	}
}

// Build base non-nitric wrapped docker image
// These will also be used for config as code runs
func BuildBaseImages(s *project.Project) error {
	errs, _ := errgroup.WithContext(context.Background())
	// set concurrent build limit here

	maxConcurrency := lo.Min([]int{goruntime.GOMAXPROCS(0), goruntime.NumCPU()})

	maxConcurrencyEnv := os.Getenv("MAX_BUILD_CONCURRENCY")
	if maxConcurrencyEnv != "" {
		newVal, err := strconv.Atoi(maxConcurrencyEnv)
		if err != nil {
			return fmt.Errorf("invalid value for MAX_BUILD_CONCURRENCY must be int got %s", maxConcurrencyEnv)
		}

		maxConcurrency = newVal
	}

	// check functions for valid names
	for _, fun := range s.Functions {
		_, err := reference.Parse(fun.Name)
		if err != nil {
			return fmt.Errorf("invalid handler name \"%s\". Names can only include alphanumeric characters, underscores, periods and hyphens", fun.Handler)
		}
	}

	pterm.Debug.Printfln("running builds %d at a time", maxConcurrency)

	errs.SetLimit(maxConcurrency)

	for _, fun := range s.Functions {
		errs.Go(buildFunction(s, fun))
	}

	return errs.Wait()
}
