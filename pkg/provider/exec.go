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

package provider

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkg/iox"
	"github.com/nitrictech/cli/pkg/netx"
)

// ProviderProcess - A deployment engine based on a locally executable binary file
type ProviderProcess struct {
	providerPath string
	process      *os.Process
	envMap       map[string]string
	Address      string
	stdout       chan<- string
	stderr       chan<- string
}

func (p *ProviderProcess) startProcess() error {
	cmd := exec.Command(p.providerPath)

	if p.envMap == nil {
		p.envMap = map[string]string{}
	}

	lis, err := netx.GetNextListener()
	if err != nil {
		return err
	}

	tcpAddr := lis.Addr().(*net.TCPAddr)

	// Set a random available port
	p.Address = lis.Addr().String()

	// TODO: consider prefixing with NITRIC_ to avoid collisions
	p.envMap["PORT"] = fmt.Sprint(tcpAddr.Port)

	if len(p.envMap) > 0 {
		env := os.Environ()

		for k, v := range p.envMap {
			env = append(env, k+"="+v)
		}

		cmd.Env = env
	}

	err = lis.Close()
	if err != nil {
		return err
	}

	cmd.Stderr = io.Discard
	cmd.Stdout = io.Discard

	if p.stderr != nil {
		cmd.Stderr = iox.NewChannelWriter(p.stderr)
	}

	if p.stdout != nil {
		cmd.Stdout = iox.NewChannelWriter(p.stdout)
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	p.process = cmd.Process

	return nil
}

func (b *ProviderProcess) Stop() error {
	if b.process != nil {
		err := b.process.Kill()
		if err != nil {
			return fmt.Errorf("failed to stop provider: %w", err)
		}
	}
	return nil
}

// isExecAny - Check if a file is executable by any user
func isExecAny(mode os.FileMode) bool {
	os := runtime.GOOS

	// could check ext in future for windows
	if os == "windows" {
		return mode.IsRegular()
	}

	return mode.IsRegular() && (mode.Perm()&0o111) != 0
}

type ProviderExecutableOption = func(*ProviderProcess)

func WithStdout(stdout chan<- string) ProviderExecutableOption {
	return func(pp *ProviderProcess) {
		pp.stdout = stdout
	}
}

func WithStderr(stderr chan<- string) ProviderExecutableOption {
	return func(pp *ProviderProcess) {
		pp.stderr = stderr
	}
}

func StartProviderExecutable(fs afero.Fs, executablePath string, opts ...ProviderExecutableOption) (*ProviderProcess, error) {
	fileInfo, err := fs.Stat(executablePath)
	if err != nil {
		return nil, err
	}

	// Ensure the file is executable
	if !isExecAny(fileInfo.Mode()) {
		return nil, fmt.Errorf("provider binary is not executable")
	}

	provProc := &ProviderProcess{
		providerPath: executablePath,
		envMap:       map[string]string{},
	}

	for _, o := range opts {
		o(provProc)
	}

	if err := provProc.startProcess(); err != nil {
		return nil, err
	}

	// return a valid binary deployment
	return provProc, nil
}
