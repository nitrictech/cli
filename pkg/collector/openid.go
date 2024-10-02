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
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

type OpenIdConfig struct {
	Issuer        string `json:"issuer"`
	JwksUri       string `json:"jwks_uri"`
	TokenEndpoint string `json:"token_endpoint"`
	AuthEndpoint  string `json:"authorization_endpoint"`
}

func validateOpenIdConnectConfig(rawUrl string) error {
	// append well-known configuration to issuer
	openIdConnectUrl, err := url.Parse(rawUrl)
	if err != nil {
		return err
	}

	timeout := 10 * time.Second
	client := http.Client{
		Timeout: timeout,
	}

	// get the configuration document
	resp, err := client.Get(openIdConnectUrl.String())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("OIDC provider failed to respond with config within %s", timeout.String())
		}

		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("received non 200 status retrieving openid-configuration: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	oidConf := &OpenIdConfig{}

	if err := json.Unmarshal(body, oidConf); err != nil {
		return errors.WithMessage(err, "error unmarshalling open id config")
	}

	// Validate that all endpoints are valid
	if _, err = url.ParseRequestURI(oidConf.AuthEndpoint); err != nil {
		return errors.WithMessage(err, "invalid auth endpoint")
	}

	if _, err = url.ParseRequestURI(oidConf.Issuer); err != nil {
		return errors.WithMessage(err, "invalid issuer")
	}

	if _, err = url.ParseRequestURI(oidConf.JwksUri); err != nil {
		return errors.WithMessage(err, "invalid jwks uri")
	}

	if _, err = url.ParseRequestURI(oidConf.TokenEndpoint); err != nil {
		return errors.WithMessage(err, "invalid token endpoint")
	}

	return nil
}
