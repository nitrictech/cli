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

package run

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/avast/retry-go"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	s3_service "github.com/nitrictech/nitric/cloud/aws/runtime/storage"
	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
	"github.com/nitrictech/nitric/core/pkg/plugins/storage"
	"github.com/nitrictech/nitric/core/pkg/worker/pool"
)

type RunStorageService struct {
	storage.StorageService
	pool pool.WorkerPool
	client *s3.Client
}

func (r *RunStorageService) ensureBucketExists(ctx context.Context, bucket string) error {
	err := retry.Do(func() error {
		_, err := r.client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucket),
		})

		return err
	}, retry.Delay(time.Second), retry.RetryIf(func(err error) bool {
		// wait for the service to become available
		return errors.Is(err, syscall.ECONNREFUSED)
	}))
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			_, err = r.client.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket:           aws.String(bucket),
				GrantFullControl: aws.String("*"),
			})
		}
	}

	return err
}

func (r *RunStorageService) triggerBucketNotification(ctx context.Context, bucket, key, eventType string) error {
	evt := &v1.TriggerRequest{
		Context: &v1.TriggerRequest_Notification{
			Notification: &v1.NotificationTriggerContext{
				Type:     v1.NotificationType_Bucket,
				Resource: bucket,
				Attributes: map[string]string{
					"key": key,
					"type": eventType,
				},
			},
		},
	}

	worker, err := r.pool.GetWorker(&pool.GetWorkerOptions{
		Trigger: evt,
	})
	if err != nil {
		// There is no worker for this notification
		return nil
	}

	if _, err := worker.HandleTrigger(ctx, evt); err != nil {
		return fmt.Errorf("error occcured triggering bucket notification: %v", err)
	}

	return nil
}

func (r *RunStorageService) Read(ctx context.Context, bucket string, key string) ([]byte, error) {
	err := r.ensureBucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}

	return r.StorageService.Read(ctx, bucket, key)
}

func (r *RunStorageService) Write(ctx context.Context, bucket string, key string, object []byte) error {
	err := r.ensureBucketExists(ctx, bucket)
	if err != nil {
		return err
	}

	err = r.StorageService.Write(ctx, bucket, key, object)
	if err != nil {
		return err
	}

	err = r.triggerBucketNotification(ctx, bucket, key, "created")
	if err != nil {
		fmt.Println(err.Error())
	}

	return nil
}

func (r *RunStorageService) Delete(ctx context.Context, bucket string, key string) error {
	err := r.ensureBucketExists(ctx, bucket)
	if err != nil {
		return err
	}

	err = r.StorageService.Delete(ctx, bucket, key)
	if err != nil {
		return err
	}

	err = r.triggerBucketNotification(ctx, bucket, key, "deleted")
	if err != nil {
		fmt.Println(err.Error())
	}

	return nil
}

func (r *RunStorageService) ListFiles(ctx context.Context, bucket string) ([]*storage.FileInfo, error) {
	err := r.ensureBucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}

	return r.StorageService.ListFiles(ctx, bucket)
}

func (r *RunStorageService) PreSignUrl(ctx context.Context, bucket string, key string, operation storage.Operation, expiry uint32) (string, error) {
	err := r.ensureBucketExists(ctx, bucket)
	if err != nil {
		return "", err
	}

	return r.StorageService.PreSignUrl(ctx, bucket, key, operation, expiry)
}

func nameSelector(nitricName string) (*string, error) {
	return &nitricName, nil
}

type StorageOptions struct {
	AccessKey string
	SecretKey string
	Endpoint  string
}

func NewStorage(pool pool.WorkerPool, opts StorageOptions) (storage.StorageService, error) {
	cfg, sessionError := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(opts.AccessKey, opts.SecretKey, "")),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: opts.Endpoint}, nil
		})),
	)
	if sessionError != nil {
		return nil, fmt.Errorf("error creating new AWS session %w", sessionError)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	})

	s3PSClient := s3.NewPresignClient(s3Client)

	s3Service, err := s3_service.NewWithClient(nil, s3Client, s3PSClient, s3_service.WithSelector(nameSelector))
	if err != nil {
		return nil, err
	}

	return &RunStorageService{
		StorageService: s3Service,
		pool: pool,
		client:         s3Client,
	}, nil
}
