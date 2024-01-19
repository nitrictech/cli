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

package storage

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/avast/retry-go"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/nitrictech/cli/pkgplus/eventbus"
	s3_service "github.com/nitrictech/nitric/cloud/aws/runtime/storage"

	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
)

type BucketName = string

type State = map[BucketName]int

// LocalStorageService - A local implementation of the storage and listeners services, bypasses the gateway to forward storage change events directly to listeners.
type LocalStorageService struct {
	client *s3.Client
	storagepb.StorageServer
	listenersLock   sync.RWMutex
	listeners       State
	server          *SeaweedServer
	storageEndpoint string

	bus EventBus.Bus
}

var (
	_ storagepb.StorageServer         = (*LocalStorageService)(nil)
	_ storagepb.StorageListenerServer = (*LocalStorageService)(nil)
)

const localStorageTopic = "local_storage"

func (s *LocalStorageService) SubscribeToState(fn func(State)) {
	s.bus.Subscribe(localStorageTopic, fn)
}

func (s *LocalStorageService) GetStorageEndpoint() string {
	return s.storageEndpoint
}

func (r *LocalStorageService) registerListener(registrationRequest *storagepb.RegistrationRequest) {
	r.listenersLock.Lock()
	defer r.listenersLock.Unlock()

	if _, ok := r.listeners[registrationRequest.BucketName]; !ok {
		r.listeners[registrationRequest.BucketName] = 0
	}

	r.listeners[registrationRequest.BucketName]++

	r.bus.Publish(localStorageTopic, r.listeners)
}

func (r *LocalStorageService) WorkerCount() int {
	r.listenersLock.RLock()
	defer r.listenersLock.RUnlock()

	workerCount := 0
	for _, val := range r.listeners {
		workerCount += val
	}

	return workerCount
}

func (r *LocalStorageService) unregisterListener(registrationRequest *storagepb.RegistrationRequest) {
	r.listenersLock.Lock()
	defer r.listenersLock.Unlock()

	r.listeners[registrationRequest.BucketName]--

	r.bus.Publish(localStorageTopic, r.listeners)
}

func (r *LocalStorageService) GetListeners() map[BucketName]int {
	r.listenersLock.RLock()
	defer r.listenersLock.RUnlock()

	return r.listeners
}

func (r *LocalStorageService) HandleRequest(req *storagepb.ServerMessage) (*storagepb.ClientMessage, error) {
	// XXX: This should not be called during local simulation
	return nil, fmt.Errorf("UNIMPLEMENTED in run storage service")
}

func (r *LocalStorageService) StopSeaweed() error {
	return r.server.Stop()
}

func (r *LocalStorageService) Listen(stream storagepb.StorageListener_ListenServer) error {
	firstRequest, err := stream.Recv()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		// first request MUST be a registration request
		return fmt.Errorf("expected registration request on first request")
	}

	stream.Send(&storagepb.ServerMessage{
		Id: firstRequest.Id,
		Content: &storagepb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &storagepb.RegistrationResponse{},
		},
	})

	bucketName := firstRequest.GetRegistrationRequest().GetBucketName()
	listenEvtType := firstRequest.GetRegistrationRequest().GetBlobEventType().String()

	listenTopicName := fmt.Sprintf("%s:%s", bucketName, listenEvtType)

	eventbus.StorageBus().SubscribeAsync(listenTopicName, func(req *storagepb.ServerMessage) {
		err := stream.Send(req)
		if err != nil {
			fmt.Println("problem sending the event")
		}
	}, false)

	// block here...
	for {
		_, err := stream.Recv()
		if err != nil {
			return err
		}

		// responses are not logged since the buckets can be viewed to review the state
	}
}

func (r *LocalStorageService) ensureBucketExists(ctx context.Context, bucket string) error {
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

func (r *LocalStorageService) triggerBucketNotifications(ctx context.Context, bucket string, key string, eventType storagepb.BlobEventType) {
	eventbus.StorageBus().Publish(fmt.Sprintf("%s:%s", bucket, eventType.String()), &storagepb.ServerMessage{
		Content: &storagepb.ServerMessage_BlobEventRequest{
			BlobEventRequest: &storagepb.BlobEventRequest{
				BucketName: bucket,
				Event: &storagepb.BlobEventRequest_BlobEvent{
					BlobEvent: &storagepb.BlobEvent{
						Key:  key,
						Type: eventType,
					},
				},
			},
		},
	})
}

// TODO: If we move declare here, we can stop attempting to lazily create buckets in the storage service
func (r *LocalStorageService) Read(ctx context.Context, req *storagepb.StorageReadRequest) (*storagepb.StorageReadResponse, error) {
	err := r.ensureBucketExists(ctx, req.BucketName)
	if err != nil {
		return nil, err
	}

	return r.StorageServer.Read(ctx, req)
}

func (r *LocalStorageService) Write(ctx context.Context, req *storagepb.StorageWriteRequest) (*storagepb.StorageWriteResponse, error) {
	err := r.ensureBucketExists(ctx, req.BucketName)
	if err != nil {
		return nil, err
	}

	resp, err := r.StorageServer.Write(ctx, req)
	if err != nil {
		return nil, err
	}

	go r.triggerBucketNotifications(ctx, req.BucketName, req.Key, storagepb.BlobEventType_Created)

	return resp, nil
}

func (r *LocalStorageService) Delete(ctx context.Context, req *storagepb.StorageDeleteRequest) (*storagepb.StorageDeleteResponse, error) {
	err := r.ensureBucketExists(ctx, req.BucketName)
	if err != nil {
		return nil, err
	}

	resp, err := r.StorageServer.Delete(ctx, req)
	if err != nil {
		return nil, err
	}

	go r.triggerBucketNotifications(ctx, req.BucketName, req.Key, storagepb.BlobEventType_Deleted)

	return resp, nil
}

func (r *LocalStorageService) ListBlobs(ctx context.Context, req *storagepb.StorageListBlobsRequest) (*storagepb.StorageListBlobsResponse, error) {
	err := r.ensureBucketExists(ctx, req.BucketName)
	if err != nil {
		return nil, err
	}

	return r.StorageServer.ListBlobs(ctx, req)
}

func (r *LocalStorageService) PreSignUrl(ctx context.Context, req *storagepb.StoragePreSignUrlRequest) (*storagepb.StoragePreSignUrlResponse, error) {
	err := r.ensureBucketExists(ctx, req.BucketName)
	if err != nil {
		return nil, err
	}

	return r.StorageServer.PreSignUrl(ctx, req)
}

func nameSelector(nitricName string) (*string, error) {
	return &nitricName, nil
}

type StorageOptions struct {
	AccessKey string
	SecretKey string
}

func NewLocalStorageService(opts StorageOptions) (*LocalStorageService, error) {
	// Start the local S3 compatible server (Seaweed)
	seaweedServer, err := NewSeaweed()
	if err != nil {
		return nil, err
	}

	err = seaweedServer.Start()
	if err != nil {
		return nil, err
	}

	storageEndpoint := fmt.Sprintf("http://localhost:%d", seaweedServer.GetApiPort())

	// Connect the S3 client to the local seaweed service
	cfg, sessionError := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(opts.AccessKey, opts.SecretKey, "")),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: storageEndpoint}, nil
		})),
		config.WithRetryMaxAttempts(5),
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

	return &LocalStorageService{
		StorageServer:   s3Service,
		client:          s3Client,
		server:          seaweedServer,
		storageEndpoint: storageEndpoint,
	}, nil
}
