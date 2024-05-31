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

package batch

import (
	"context"
	"fmt"

	batchpb "github.com/nitrictech/nitric/core/pkg/proto/batch/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/jobs"
)

type BatchRunner func(req *batchpb.JobSubmitRequest) error

type LocalBatchService struct {
	*jobs.JobManager
	batchpb.UnimplementedBatchServer
}

var (
	_ batchpb.BatchServer = (*LocalBatchService)(nil)
	_ batchpb.JobServer   = (*LocalBatchService)(nil)
)

func (l *LocalBatchService) SubmitJob(ctx context.Context, req *batchpb.JobSubmitRequest) (*batchpb.JobSubmitResponse, error) {
	// TODO: Error if job does not exist
	// Execute the job request in the background
	go func() {
		_, err := l.HandleJobRequest(&batchpb.ServerMessage{
			Content: &batchpb.ServerMessage_JobRequest{
				JobRequest: &batchpb.JobRequest{
					JobName: req.GetJobName(),
					Data:    req.Data,
				},
			},
		})
		if err != nil {
			// TODO: Log error correctly
			fmt.Println("Error handling job request: ", err)
		}
	}()

	return &batchpb.JobSubmitResponse{}, nil
}

func NewLocalBatchService() *LocalBatchService {
	return &LocalBatchService{
		JobManager: jobs.New(),
	}
}
