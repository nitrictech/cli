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

package common

import (
	"bytes"
	"encoding/json"
	"io"
)

type PulumiResource struct {
	ID      string                 `json:"id"`
	URN     string                 `json:"urn"`
	Type    string                 `json:"type"`
	Outputs map[string]interface{} `json:"outputs"`
}

type PulumiLatest struct {
	Resources []PulumiResource `json:"resources"`
}

type PulumiCheckpoint struct {
	Stack  string       `json:"stack"`
	Latest PulumiLatest `json:"latest"`
}

type PulumiStack struct {
	Checkpoint PulumiCheckpoint `json:"checkpoint"`
}

func StackFromReader(r io.ReadCloser) (*PulumiStack, error) {
	b := &bytes.Buffer{}

	_, err := io.Copy(b, r)
	if err != nil {
		return nil, err
	}

	stack := &PulumiStack{}
	err = json.Unmarshal(b.Bytes(), &stack)

	return stack, err
}
