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

package codeconfig

import (
	"fmt"
	"net"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	v1 "github.com/nitrictech/apis/go/nitric/v1"
	"github.com/nitrictech/newcli/pkg/containerengine"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/utils"
)

// CodeConfig - represents a collection of related functions and their shared dependencies.
type CodeConfig interface {
	Collect() error
	ImagesToBuild() map[string]string
	ToStack() (*stack.Stack, error)
}

type codeConfig struct {
	// A stack can be composed of one or more applications
	functions map[string]*FunctionDependencies
	stackPath string
	files     []string
	lock      sync.RWMutex
}

func New(stackPath string, globString string) (CodeConfig, error) {
	files, err := filepath.Glob(filepath.Join(stackPath, globString))
	if err != nil {
		return nil, err
	}

	return &codeConfig{
		stackPath: stackPath,
		files:     files,
		functions: map[string]*FunctionDependencies{},
		lock:      sync.RWMutex{},
	}, nil
}

func (c *codeConfig) ImagesToBuild() map[string]string {
	imagesToBuild := map[string]string{}
	for _, h := range c.files {
		lang := strings.Replace(path.Ext(h), ".", "", 1)
		imagesToBuild[lang] = imageNameFromExt(path.Ext(h))
	}
	return imagesToBuild
}

func imageNameFromExt(ext string) string {
	return "nitric-" + strings.Replace(ext, ".", "", 1) + "-dev"
}

// Collect - Collects information about all functions for a nitric stack
func (c *codeConfig) Collect() error {
	for _, f := range c.files {
		rel, err := filepath.Rel(c.stackPath, f)
		if err != nil {
			return err
		}

		err = c.collectOne(rel)
		if err != nil {
			return err
		}
	}
	return nil
}

// apiSpec produces an open api v3 spec for the requests API name
func (c *codeConfig) apiSpec(api string) (*openapi3.T, error) {
	doc := &openapi3.T{
		Paths: make(openapi3.Paths),
	}

	doc.Info = &openapi3.Info{
		Title:   api,
		Version: "v1",
	}

	doc.OpenAPI = "3.0.1"

	// Compile an API specification from the functions in the stack for the given API name
	workers := make([]*v1.ApiWorker, 0)

	// Collect all workers
	for _, f := range c.functions {
		workers = append(workers, f.apis[api].workers...)
	}

	// loop over workers to build new api specification
	// FIXME: We will need to merge path matches across all workers
	// to ensure we don't have conflicts
	for _, w := range workers {
		params := make(openapi3.Parameters, 0)
		normalizedPath := ""
		for _, p := range strings.Split(w.Path, "/") {
			if strings.HasPrefix(p, ":") {
				paramName := strings.Replace(p, ":", "", -1)
				params = append(params, &openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						In:   "path",
						Name: paramName,
					},
				})
				normalizedPath = normalizedPath + "{" + paramName + "}" + "/"
			} else {
				normalizedPath = normalizedPath + p + "/"
			}
		}

		pathItem := doc.Paths.Find(normalizedPath)

		if pathItem == nil {
			// Add the parameters at the path level
			pathItem = &openapi3.PathItem{
				Parameters: params,
			}
			// Add the path item to the document
			doc.Paths[normalizedPath] = pathItem
		}

		for _, m := range w.Methods {
			if pathItem.Operations() != nil && pathItem.Operations()[m] != nil {
				// If the operation already exists we should fail
				// NOTE: This should not happen as operations are stored in a map
				// in the api state for functions
				return nil, fmt.Errorf("found conflicting operations")
			}

			// See if the path already exists
			doc.AddOperation(normalizedPath, m, &openapi3.Operation{
				OperationID: normalizedPath + m,
				Responses:   openapi3.NewResponses(),
			})
		}
	}

	return doc, nil
}

// collectOne - Collects information about a function for a nitric stack
// handler - the specific handler for the application
func (c *codeConfig) collectOne(handler string) error {
	fun := NewFunction()
	srv := NewServer(fun)
	grpcSrv := grpc.NewServer()

	v1.RegisterResourceServiceServer(grpcSrv, srv)
	v1.RegisterFaasServiceServer(grpcSrv, srv)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}

	defer lis.Close()

	errChan := make(chan error)
	go func(errChan chan error) {
		errChan <- grpcSrv.Serve(lis)
	}(errChan)

	// run the handler in a container
	// Specify the service bind as the port with the docker gateway IP (running in bridge mode)
	ce, err := containerengine.Discover()
	if err != nil {
		return errors.WithMessage(err, "error running the handler")
	}

	hostConfig := &container.HostConfig{
		AutoRemove: true,
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: c.stackPath,
				Target: "/app",
			},
		},
	}
	if runtime.GOOS == "linux" {
		// setup host.docker.internal to route to host gateway
		// to access rpc server hosted by local CLI run
		hostConfig.ExtraHosts = []string{"host.docker.internal:172.17.0.1"}
	}

	cID, err := ce.ContainerCreate(&container.Config{
		Image: imageNameFromExt(path.Ext(handler)), // Select an image to use based on the handler
		// Set the address to the bound port
		Env: []string{"SERVICE_ADDRESS=host.docker.internal:50051"},
		Cmd: strslice.StrSlice{"-T", handler},
	}, hostConfig, nil, containerNameFromHandler(handler))
	if err != nil {
		return err
	}

	err = ce.Start(cID)
	if err != nil {
		return err
	}

	errs := utils.NewErrorList()
	waitChan, cErrChan := ce.ContainerWait(cID, container.WaitConditionNextExit)
	select {
	case done := <-waitChan:
		msg := ""
		if done.Error != nil {
			msg = done.Error.Message
		}
		if msg != "" || done.StatusCode != 0 {
			errs.Add(fmt.Errorf("error executing container (code %d) %s", done.StatusCode, msg))
		}
	case cErr := <-cErrChan:
		errs.Add(cErr)
	}

	// When the container exits stop the server
	grpcSrv.Stop()
	errs.Add(<-errChan)

	// Add the function
	c.addFunction(fun, handler)
	return errs.Aggregate()
}

func containerNameFromHandler(handler string) string {
	return strings.Replace(path.Base(handler), path.Ext(handler), "", 1)
}

func (c *codeConfig) addFunction(fun *FunctionDependencies, handler string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.functions[handler] = fun
}

func (c *codeConfig) ToStack() (*stack.Stack, error) {
	s := &stack.Stack{
		Name:        path.Base(c.stackPath),
		Functions:   map[string]stack.Function{},
		Containers:  map[string]stack.Container{},
		Collections: map[string]interface{}{},
		Buckets:     map[string]interface{}{},
		Topics:      map[string]interface{}{},
		Queues:      map[string]interface{}{},
		Schedules:   map[string]stack.Schedule{},
		Apis:        map[string]string{},
		Sites:       map[string]stack.Site{},
		EntryPoints: map[string]stack.Entrypoint{},
	}
	errs := utils.NewErrorList()
	for handler, f := range c.functions {
		name := strings.Replace(path.Base(handler), path.Ext(handler), "", 1)
		s.Functions[name] = stack.Function{Handler: handler}
		for k, v := range f.apis {
			if current, ok := s.Apis[k]; ok {
				if current != v.workers[0].String() {
					errs.Add(fmt.Errorf("API %s has mulitple values %s %s", k, current, v.workers[0].String()))
				}
			} else {
				s.Apis[k] = v.workers[0].String()
			}
		}
		for k, v := range f.buckets {
			if current, ok := s.Buckets[k]; ok {
				if current != v.String() {
					errs.Add(fmt.Errorf("bucket %s has mulitple values %s %s", k, current, v.String()))
				}
			} else {
				s.Buckets[k] = v.String()
			}
		}
		for k, v := range f.collections {
			if current, ok := s.Collections[k]; ok {
				if current != v.String() {
					errs.Add(fmt.Errorf("collection %s has mulitple values %s %s", k, current, v.String()))
				}
			} else {
				s.Collections[k] = v.String()
			}
		}
		for k, v := range f.queues {
			if current, ok := s.Queues[k]; ok {
				if current != v.String() {
					errs.Add(fmt.Errorf("queue %s has mulitple values %s %s", k, current, v.String()))
				}
			} else {
				s.Queues[k] = v.String()
			}
		}
		for k, v := range f.schedules {
			newS := stack.Schedule{
				Expression: v.String(),
				Target:     stack.ScheduleTarget{},
				Event:      stack.ScheduleEvent{},
			}
			if current, ok := s.Schedules[k]; ok {
				if err := mergo.Merge(&current, &newS); err != nil {
					errs.Add(err)
				} else {
					s.Schedules[k] = current
				}
			} else {
				s.Schedules[k] = newS
			}
		}
		for k, v := range f.topics {
			if current, ok := s.Topics[k]; ok {
				if current != v.String() {
					errs.Add(fmt.Errorf("topic %s has mulitple values %s %s", k, current, v.String()))
				}
			} else {
				s.Topics[k] = v.String()
			}
		}
	}

	return s, errs.Aggregate()
}
