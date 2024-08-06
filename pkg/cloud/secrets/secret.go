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
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/nitrictech/cli/pkg/cloud/env"
	grpc_errors "github.com/nitrictech/nitric/core/pkg/grpc/errors"
	secretspb "github.com/nitrictech/nitric/core/pkg/proto/secrets/v1"
)

type DevSecretService struct {
	secDir string
	mu     sync.RWMutex
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
	file.Close()

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
	latestFile.Close()

	return &secretspb.SecretPutResponse{
		SecretVersion: &secretspb.SecretVersion{
			Secret:  req.Secret,
			Version: versionId,
		},
	}, nil
}

func (s *DevSecretService) Access(ctx context.Context, req *secretspb.SecretAccessRequest) (*secretspb.SecretAccessResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	newErr := grpc_errors.ErrorsWithScope(
		"DevSecretService.Access",
	)

	content, err := os.ReadFile(s.secretFileName(req.SecretVersion.Secret, req.SecretVersion.Version))
	if err != nil {
		// If the file is missing it's typically because it hasn't been created yet
		if os.IsNotExist(err) {
			return nil, newErr(
				codes.NotFound,
				"failed to retrieve secret value, ensure a value has been stored using the `put` method, before attempting to access it",
				err,
			)
		}

		return nil, newErr(
			codes.Unknown,
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

type SecretVersion struct {
	Version   string `json:"version"`
	Value     string `json:"value"`
	Latest    bool   `json:"latest"`
	CreatedAt string `json:"createdAt"`
}

// List all secret versions and values for a given secret, used by dashboard
func (s *DevSecretService) List(ctx context.Context, secretName string) ([]SecretVersion, error) {
	newErr := grpc_errors.ErrorsWithScope(
		"DevSecretService.List",
	)

	// Check whether file exists
	_, err := os.Stat(s.secDir)
	if os.IsNotExist(err) {
		return nil, newErr(codes.NotFound, "secret store not found", err)
	}

	// List all files in the directory
	files, err := os.ReadDir(s.secDir)
	if err != nil {
		return nil, newErr(codes.FailedPrecondition, "error reading secret store", err)
	}

	// Create a response
	resp := []SecretVersion{}

	var latestVersion SecretVersion

	for _, file := range files {
		// Check whether the file is a secret file
		if strings.HasSuffix(file.Name(), ".txt") {
			// Split the file name to get the secret name and version
			splitName := strings.Split(file.Name(), "_")
			// Check whether the secret name matches the requested secret
			if splitName[0] == secretName {
				version := strings.TrimSuffix(splitName[1], ".txt")

				info, err := file.Info()
				if err != nil {
					return nil, newErr(codes.FailedPrecondition, "error reading file info", err)
				}

				createdAt := info.ModTime().Format("2006-01-02 15:04:05")

				valueResp, err := s.Access(ctx, &secretspb.SecretAccessRequest{
					SecretVersion: &secretspb.SecretVersion{
						Secret:  &secretspb.Secret{Name: secretName},
						Version: version,
					},
				})
				if err != nil {
					// check if not found and add blank value
					if strings.HasPrefix(err.Error(), "rpc error: code = NotFound desc") {
						resp = append(resp, SecretVersion{
							Version:   version,
							Value:     "",
							CreatedAt: createdAt,
						})

						continue
					}

					return nil, newErr(codes.FailedPrecondition, "error reading version value", err)
				}

				// Check whether the version is the latest
				if version == "latest" {
					latestVersion = SecretVersion{
						Value:     string(valueResp.Value),
						CreatedAt: createdAt,
					}

					continue
				}

				// Add the secret to the response
				resp = append(resp, SecretVersion{
					Version:   version,
					Value:     string(valueResp.Value),
					CreatedAt: createdAt,
				})
			}
		}
	}

	if len(resp) > 0 {
		// sort by created at
		sort.Slice(resp, func(i, j int) bool {
			return resp[i].CreatedAt > resp[j].CreatedAt
		})

		// mark latest version
		if resp[0].Value == latestVersion.Value {
			resp[0].Latest = true
		}
	}

	return resp, nil
}

// Delete a secret version, used by dashboard
func (s *DevSecretService) Delete(ctx context.Context, secretName string, version string, latest bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newErr := grpc_errors.ErrorsWithScope(
		"DevSecretService.Delete",
	)

	// Check whether file exists
	_, err := os.Stat(s.secDir)
	if os.IsNotExist(err) {
		return newErr(codes.NotFound, "secret store not found", err)
	}

	// delete the version file
	err = os.Remove(s.secretFileName(&secretspb.Secret{Name: secretName}, version))
	if err != nil {
		return newErr(codes.Internal, "error deleting secret version", err)
	}

	if latest {
		// delete the latest file
		err = os.Remove(s.secretFileName(&secretspb.Secret{Name: secretName}, "latest"))
		if err != nil {
			return newErr(codes.Internal, "error deleting latest secret version", err)
		}

		// get last latest version and create a new latest
		entries, err := os.ReadDir(s.secDir)
		if err != nil {
			return newErr(codes.FailedPrecondition, "error reading secret store", err)
		}

		var files []os.FileInfo

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				return newErr(codes.FailedPrecondition, "error reading file info", err)
			}

			files = append(files, info)
		}

		// sort files by date
		sort.Slice(files, func(i, j int) bool {
			return files[i].ModTime().Before(files[j].ModTime())
		})

		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".txt") {
				splitName := strings.Split(file.Name(), "_")
				if splitName[0] == secretName {
					version := strings.TrimSuffix(splitName[1], ".txt")

					// copy file as new latest file with same contents
					destinationFile, err := os.Create(s.secretFileName(&secretspb.Secret{Name: secretName}, "latest"))
					if err != nil {
						return newErr(codes.FailedPrecondition, "error creating latest secret version", err)
					}

					sourceFile, err := os.Open(s.secretFileName(&secretspb.Secret{Name: secretName}, version))
					if err != nil {
						return newErr(codes.FailedPrecondition, "error reading secret version", err)
					}

					_, err = io.Copy(destinationFile, sourceFile)
					if err != nil {
						return newErr(codes.FailedPrecondition, "error copying secret version", err)
					}

					err = destinationFile.Sync()
					if err != nil {
						return newErr(codes.FailedPrecondition, "error syncing latest secret version", err)
					}

					sourceFile.Close()
					destinationFile.Close()
				}
			}
		}
	}

	return nil
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
