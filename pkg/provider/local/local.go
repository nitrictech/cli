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

package local

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/nitrictech/newcli/pkg/containerengine"
	"github.com/nitrictech/newcli/pkg/provider/types"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/target"
)

const (
	devVolume        = "/nitric/"
	runDir           = "./.nitric/run"
	runPerm          = 0o777 // NOTE: octal notation is important here!!!
	LabelRunID       = "io.nitric-run-id"
	LabelStackName   = "io.nitric-stack"
	LabelType        = "io.nitric-type"
	minioPort        = 9000
	minioConsolePort = 9001 // TODO: Determine if we would like to expose the console
)

var (
	userHome, _   = os.UserHomeDir()
	nitricHome    = path.Join(userHome, ".nitric")
	stagingDir    = path.Join(nitricHome, "staging")
	stagingAPIDir = path.Join(stagingDir, "apis")
)

type local struct {
	s       *stack.Stack
	t       *target.Target
	network string
	cr      containerengine.ContainerEngine
}

func New(s *stack.Stack, t *target.Target) (types.Provider, error) {
	cr, err := containerengine.Discover()
	if err != nil {
		return nil, err
	}

	return &local{
		s:       s,
		t:       t,
		cr:      cr,
		network: "bridge",
	}, nil
}

func (l *local) Apply(name string) error {
	err := l.cr.RemoveByLabel(LabelStackName, l.s.Name)
	if err != nil {
		return err
	}

	l.network = fmt.Sprintf("%s-net-%s", l.s.Name, name)
	err = l.cr.NetworkCreate(l.network)
	if err != nil {
		return errors.WithMessage(err, "network")
	}

	err = l.storage(name)
	if err != nil {
		return errors.WithMessage(err, "storage")
	}

	for _, f := range l.s.Functions {
		err = l.function(name, &f)
		if err != nil {
			return errors.WithMessage(err, "function "+f.Name())
		}
	}

	for k, apiFile := range l.s.Apis {
		err = l.gateway(name, k, apiFile)
		if err != nil {
			return errors.WithMessage(err, "gateway "+k)
		}
	}

	for k, v := range l.s.EntryPoints {
		err = l.entrypoint(name, k, &v)
		if err != nil {
			return errors.WithMessage(err, "entrypoint "+k)
		}
	}
	return nil
}

type containerSummary struct {
	Image  string
	ID     string
	Type   string
	State  string
	Status string
	Ports  []int
}

func (l *local) List() (interface{}, error) {
	res, err := l.cr.ContainersListByLabel(map[string]string{LabelStackName: l.s.Name})
	if err != nil {
		return nil, err
	}
	cons := []containerSummary{}
	for _, c := range res {
		ports := []int{}
		for _, p := range c.Ports {
			ports = append(ports, int(p.PublicPort))
		}
		cons = append(cons, containerSummary{
			Image:  c.Image,
			ID:     c.ID[0:12],
			Type:   c.Labels[LabelType],
			State:  c.State,
			Status: c.Status,
			Ports:  ports,
		})
	}
	return cons, nil
}

func (l *local) Delete(name string) error {
	return l.cr.RemoveByLabel(LabelStackName, l.s.Name)
}

func (l *local) labels(deploymentName, contType string) map[string]string {
	return map[string]string{
		LabelStackName: l.s.Name,
		LabelRunID:     deploymentName,
		LabelType:      contType,
	}
}
