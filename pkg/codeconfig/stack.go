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
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"

	v1 "github.com/nitrictech/apis/go/nitric/v1"
)

// Stack - represents a collection of related functions and their shared dependencies.
type Stack struct {
	// A stack can be composed of one or more applications
	functions []*Function
	lock      sync.RWMutex
}

func (s *Stack) AddFunction(fun *Function) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.functions = append(s.functions, fun)
}

// Produce an open api v3 spec for the requests API name
func (s *Stack) GetApiSpec(api string) (*openapi3.T, error) {
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
	for _, f := range s.functions {
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

func (s *Stack) String() string {
	funcStrings := make([]string, 0)

	for _, f := range s.functions {
		funcStrings = append(funcStrings, f.String())
	}

	return fmt.Sprintf(`
	  functions: [%s]
	`, strings.Join(funcStrings, ",\n"))
}

func NewStack() *Stack {
	return &Stack{
		functions: make([]*Function, 0),
		lock:      sync.RWMutex{},
	}
}
