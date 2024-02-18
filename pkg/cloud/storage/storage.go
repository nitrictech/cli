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
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/gorilla/mux"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nitrictech/cli/pkg/cloud/env"
	"github.com/nitrictech/cli/pkg/eventbus"
	"github.com/nitrictech/cli/pkg/grpcx"

	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
)

type (
	BucketName  = string
	serviceName = string
)

type State = map[BucketName]map[serviceName]int

// LocalStorageService - A local implementation of the storage and listeners services, bypasses the gateway to forward storage change events directly to listeners.
type LocalStorageService struct {
	listenersLock sync.RWMutex
	listeners     State

	storageListener net.Listener
	storageServer   http.Server

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

func (r *LocalStorageService) registerListener(serviceName string, registrationRequest *storagepb.RegistrationRequest) {
	r.listenersLock.Lock()
	defer r.listenersLock.Unlock()

	if r.listeners[registrationRequest.BucketName] == nil {
		r.listeners[registrationRequest.BucketName] = map[string]int{}
	}

	if _, ok := r.listeners[registrationRequest.BucketName]; !ok {
		r.listeners[registrationRequest.BucketName][serviceName] = 0
	}

	r.listeners[registrationRequest.BucketName][serviceName]++

	r.bus.Publish(localStorageTopic, r.listeners)
}

func (r *LocalStorageService) WorkerCount() int {
	r.listenersLock.RLock()
	defer r.listenersLock.RUnlock()

	workerCount := 0
	for _, services := range r.listeners {
		for _, val := range services {
			workerCount += val
		}
	}

	return workerCount
}

func (r *LocalStorageService) unregisterListener(serviceName string, registrationRequest *storagepb.RegistrationRequest) {
	r.listenersLock.Lock()
	defer r.listenersLock.Unlock()

	r.listeners[registrationRequest.BucketName][serviceName]--

	r.bus.Publish(localStorageTopic, r.listeners)
}

func (r *LocalStorageService) GetListeners() map[BucketName]map[serviceName]int {
	r.listenersLock.RLock()
	defer r.listenersLock.RUnlock()

	return r.listeners
}

func (r *LocalStorageService) HandleRequest(req *storagepb.ServerMessage) (*storagepb.ClientMessage, error) {
	// XXX: This should not be called during local simulation
	return nil, fmt.Errorf("UNIMPLEMENTED in run storage service")
}

func (r *LocalStorageService) Listen(stream storagepb.StorageListener_ListenServer) error {
	serviceName, err := grpcx.GetServiceNameFromStream(stream)
	if err != nil {
		return err
	}

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

	r.registerListener(serviceName, firstRequest.GetRegistrationRequest())
	defer r.unregisterListener(serviceName, firstRequest.GetRegistrationRequest())

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
	return os.MkdirAll(filepath.Join(env.LOCAL_BUCKETS_DIR.String(), bucket), os.ModePerm)
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

	fileRef := filepath.Join(env.LOCAL_BUCKETS_DIR.String(), req.BucketName, req.Key)

	contents, err := os.ReadFile(fileRef)
	if err != nil {
		return nil, err
	}

	return &storagepb.StorageReadResponse{
		Body: contents,
	}, nil
}

func (r *LocalStorageService) Exists(ctx context.Context, req *storagepb.StorageExistsRequest) (*storagepb.StorageExistsResponse, error) {
	fileRef := filepath.Join(env.LOCAL_BUCKETS_DIR.String(), req.BucketName, req.Key)

	_, err := os.Stat(fileRef)

	if err != nil {
		if os.IsNotExist(err) {
			return &storagepb.StorageExistsResponse{
				Exists: false,
			}, nil
		}

		return nil, err
	}

	return &storagepb.StorageExistsResponse{
		Exists: true,
	}, nil
}

func (r *LocalStorageService) Write(ctx context.Context, req *storagepb.StorageWriteRequest) (*storagepb.StorageWriteResponse, error) {
	err := r.ensureBucketExists(ctx, req.BucketName)
	if err != nil {
		return nil, err
	}

	fileRef := filepath.Join(env.LOCAL_BUCKETS_DIR.String(), req.BucketName, req.Key)

	err = os.WriteFile(fileRef, req.Body, os.ModePerm)
	if err != nil {
		return nil, err
	}

	go r.triggerBucketNotifications(ctx, req.BucketName, req.Key, storagepb.BlobEventType_Created)

	return &storagepb.StorageWriteResponse{}, nil
}

func (r *LocalStorageService) Delete(ctx context.Context, req *storagepb.StorageDeleteRequest) (*storagepb.StorageDeleteResponse, error) {
	err := r.ensureBucketExists(ctx, req.BucketName)
	if err != nil {
		return nil, err
	}

	fileRef := filepath.Join(env.LOCAL_BUCKETS_DIR.String(), req.BucketName, req.Key)

	err = os.Remove(fileRef)
	if err != nil {
		return nil, err
	}

	go r.triggerBucketNotifications(ctx, req.BucketName, req.Key, storagepb.BlobEventType_Deleted)

	return &storagepb.StorageDeleteResponse{}, nil
}

func (r *LocalStorageService) ListBlobs(ctx context.Context, req *storagepb.StorageListBlobsRequest) (*storagepb.StorageListBlobsResponse, error) {
	err := r.ensureBucketExists(ctx, req.BucketName)
	if err != nil {
		return nil, err
	}
	blobs := []*storagepb.Blob{}

	localBucket := filepath.Join(env.LOCAL_BUCKETS_DIR.String(), req.BucketName)

	err = filepath.Walk(localBucket, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(localBucket, path)
			if err != nil {
				return err
			}
			blobs = append(blobs, &storagepb.Blob{
				Key: relPath,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &storagepb.StorageListBlobsResponse{
		Blobs: blobs,
	}, nil
}

func (r *LocalStorageService) PreSignUrl(ctx context.Context, req *storagepb.StoragePreSignUrlRequest) (*storagepb.StoragePreSignUrlResponse, error) {
	err := r.ensureBucketExists(ctx, req.BucketName)
	if err != nil {
		return nil, err
	}

	var address string = ""

	switch req.Operation {
	case storagepb.StoragePreSignUrlRequest_WRITE:
		address = fmt.Sprintf("http://localhost:%d/write/%s/%s", r.storageListener.Addr().(*net.TCPAddr).Port, req.BucketName, url.PathEscape(req.Key))
	case storagepb.StoragePreSignUrlRequest_READ:
		address = fmt.Sprintf("http://localhost:%d/read/%s/%s", r.storageListener.Addr().(*net.TCPAddr).Port, req.BucketName, url.PathEscape(req.Key))
	}

	if address == "" {
		status.Error(codes.Internal, "error generating presigned url")
	}

	return &storagepb.StoragePreSignUrlResponse{
		Url: address,
	}, nil
}

type StorageOptions struct {
	AccessKey string
	SecretKey string
}

func NewLocalStorageService(opts StorageOptions) (*LocalStorageService, error) {
	var err error
	storageService := &LocalStorageService{
		listeners: map[string]map[string]int{},
		bus:       EventBus.New(),
	}

	storageService.storageListener, err = net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}

	router := mux.NewRouter()

	router.HandleFunc("/read/{bucket}/{file}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bucket := vars["bucket"]
		file := vars["file"]

		resp, err := storageService.Read(context.Background(), &storagepb.StorageReadRequest{
			BucketName: bucket,
			Key:        file,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(resp.Body)
	})

	router.HandleFunc("/write/{bucket}/{file}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bucket := vars["bucket"]
		file := vars["file"]

		content, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = storageService.Write(context.Background(), &storagepb.StorageWriteRequest{
			BucketName: bucket,
			Key:        file,
			Body:       content,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	go http.Serve(storageService.storageListener, router)

	return storageService, nil
}
