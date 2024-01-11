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

package secrets

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/nitrictech/cli/pkgplus/cloud/env"
	grpc_errors "github.com/nitrictech/nitric/core/pkg/grpc/errors"
	secretspb "github.com/nitrictech/nitric/core/pkg/proto/secrets/v1"
)

type DevSecretService struct {
	secDir string
}

var _ secretspb.SecretManagerServer = (*DevSecretService)(nil)

func (s *DevSecretService) secretFileName(sec *secretspb.Secret, v string) string {
	filename := fmt.Sprintf("%s_%s.txt", sec.Name, v)
	return filepath.Join(s.secDir, filename)
}

func (s *DevSecretService) Put(ctx context.Context, req *secretspb.SecretPutRequest) (*secretspb.SecretPutResponse, error) {
	newErr := grpc_errors.ErrorsWithScope(
		"DevSecretService.Put",
	)

	versionId := uuid.New().String()
	// Creates a new file in the form:
	// DIR/Name_Version.txt
	file, err := os.Create(s.secretFileName(req.Secret, versionId))
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"error creating secret store",
			err,
		)
	}

	sVal := base64.StdEncoding.EncodeToString(req.Value)

	writer := bufio.NewWriter(file)

	_, err = writer.WriteString(sVal)
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"error writing secret value",
			err,
		)
	}

	writer.Flush()

	// Creates a new file as latest
	latestFile, err := os.Create(s.secretFileName(req.Secret, "latest"))
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"error creating latest secret",
			err,
		)
	}

	latestWriter := bufio.NewWriter(latestFile)

	_, err = latestWriter.WriteString(sVal + "," + versionId)
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"error writing secret value",
			err,
		)
	}

	latestWriter.Flush()

	return &secretspb.SecretPutResponse{
		SecretVersion: &secretspb.SecretVersion{
			Secret:  req.Secret,
			Version: versionId,
		},
	}, nil
}

func (s *DevSecretService) Access(ctx context.Context, req *secretspb.SecretAccessRequest) (*secretspb.SecretAccessResponse, error) {
	newErr := grpc_errors.ErrorsWithScope(
		"DevSecretService.Access",
	)

	content, err := os.ReadFile(s.secretFileName(req.SecretVersion.Secret, req.SecretVersion.Version))
	if err != nil {
		return nil, newErr(
			codes.InvalidArgument,
			"error reading secret store",
			err,
		)
	}

	splitContent := strings.Split(string(content), ",")
	version := req.SecretVersion.Version
	// check whether a version number is stored in the file, this indicates the 'latest' version file.
	if len(splitContent) == 2 {
		version = splitContent[1]
	}

	sVal, err := base64.StdEncoding.DecodeString(splitContent[0])
	if err != nil {
		return nil, err
	}

	return &secretspb.SecretAccessResponse{
		SecretVersion: &secretspb.SecretVersion{
			Secret:  req.SecretVersion.Secret,
			Version: version,
		},
		Value: sVal,
	}, nil
}

// Create new secret store
func NewSecretService() (*DevSecretService, error) {
	secDir := env.LOCAL_SECRETS_DIR.String()
	// Check whether file exists
	_, err := os.Stat(secDir)
	if os.IsNotExist(err) {
		// Make directory if not present
		err := os.MkdirAll(secDir, 0o777)
		if err != nil {
			return nil, err
		}
	}

	return &DevSecretService{
		secDir: secDir,
	}, nil
}
