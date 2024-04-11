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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/gorilla/mux"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/nitrictech/cli/pkg/cloud/env"
	"github.com/nitrictech/cli/pkg/eventbus"
	"github.com/nitrictech/cli/pkg/grpcx"

	"github.com/nitrictech/nitric/core/pkg/logger"
	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
)

type (
	BucketName  = string
	serviceName = string
)

type State = map[BucketName]map[serviceName]int

// Generate a signing secret for presigned URL tokens at runtime
var signingSecret *string = nil
var signingSecretLock sync.Mutex

func getSigningSecret() ([]byte, error) {
	signingSecretLock.Lock()
	defer signingSecretLock.Unlock()

	if signingSecret == nil {
		key := make([]byte, 32) // Generate a 256-bit key
		_, err := rand.Read(key)
		if err != nil {
			return nil, err
		}

		secret := base64.StdEncoding.EncodeToString(key)
		signingSecret = &secret
	}

	return []byte(*signingSecret), nil
}

// LocalStorageService - A local implementation of the storage and listeners services, bypasses the gateway to forward storage change events directly to listeners.
type LocalStorageService struct {
	listenersLock sync.RWMutex
	listeners     State

	storageListener net.Listener

	bus EventBus.Bus
}

var (
	_ storagepb.StorageServer         = (*LocalStorageService)(nil)
	_ storagepb.StorageListenerServer = (*LocalStorageService)(nil)
)

const localStorageTopic = "local_storage"

func (s *LocalStorageService) SubscribeToState(fn func(State)) {
	_ = s.bus.Subscribe(localStorageTopic, fn)
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

	err = stream.Send(&storagepb.ServerMessage{
		Id: firstRequest.Id,
		Content: &storagepb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &storagepb.RegistrationResponse{},
		},
	})
	if err != nil {
		return err
	}

	bucketName := firstRequest.GetRegistrationRequest().GetBucketName()
	listenEvtType := firstRequest.GetRegistrationRequest().GetBlobEventType().String()

	listenTopicName := fmt.Sprintf("%s:%s", bucketName, listenEvtType)

	r.registerListener(serviceName, firstRequest.GetRegistrationRequest())
	defer r.unregisterListener(serviceName, firstRequest.GetRegistrationRequest())

	err = eventbus.StorageBus().SubscribeAsync(listenTopicName, func(req *storagepb.ServerMessage) {
		err := stream.Send(req)
		if err != nil {
			fmt.Println("problem sending the event")
		}
	}, false)
	if err != nil {
		return fmt.Errorf("error subscribing to topic: %s", err.Error())
	}

	// block here...
	for {
		_, err := stream.Recv()
		if err != nil {
			return err
		} // responses are not logged since the buckets can be viewed to review the state
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

	// Ensure the directory structure exists
	err = os.MkdirAll(filepath.Dir(fileRef), os.ModePerm)
	if err != nil {
		return nil, err
	}

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
				Key: filepath.ToSlash(relPath),
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

func tokenFromRequest(req *storagepb.StoragePreSignUrlRequest) *jwt.Token {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(req.Expiry.AsDuration()).Unix(),
		"request": map[string]string{
			"bucket": req.BucketName,
			"key":    req.Key,
			"op":     req.Operation.String(),
		},
	})

}

func requestFromToken(token string) (*storagepb.StoragePreSignUrlRequest, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return getSigningSecret()
	}, jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("could not convert claims to map")
	}

	requestMap, ok := claims["request"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not convert request to map")
	}

	return &storagepb.StoragePreSignUrlRequest{
		BucketName: requestMap["bucket"].(string),
		Key:        requestMap["key"].(string),
		Operation:  storagepb.StoragePreSignUrlRequest_Operation(storagepb.StoragePreSignUrlRequest_Operation_value[requestMap["op"].(string)]),
	}, nil
}

func (r *LocalStorageService) PreSignUrl(ctx context.Context, req *storagepb.StoragePreSignUrlRequest) (*storagepb.StoragePreSignUrlResponse, error) {
	err := r.ensureBucketExists(ctx, req.BucketName)
	if err != nil {
		return nil, err
	}

	var address string = ""

	token := tokenFromRequest(req)
	secret, err := getSigningSecret()
	if err != nil {
		return nil, status.Error(codes.Internal, "error generating presigned url, could not get signing secret")
	}

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("error generating presigned url, could not sign token: %v", err))
	}

	// XXX: Do not URL encode keys (path needs to be preserved)
	// TODO: May need to re-write slashes to a non-escapable character format
	switch req.Operation {
	case storagepb.StoragePreSignUrlRequest_WRITE:
		address = fmt.Sprintf("http://localhost:%d/write/%s", r.storageListener.Addr().(*net.TCPAddr).Port, tokenString)
	case storagepb.StoragePreSignUrlRequest_READ:
		address = fmt.Sprintf("http://localhost:%d/read/%s", r.storageListener.Addr().(*net.TCPAddr).Port, tokenString)
	}

	if address == "" {
		return nil, status.Error(codes.Internal, "error generating presigned url, unknown operation")
	}

	return &storagepb.StoragePreSignUrlResponse{
		Url: address,
	}, nil
}

type StorageOptions struct {
	AccessKey string
	SecretKey string
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, PUT")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
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
	router.Use(corsMiddleware)

	router.HandleFunc("/read/{token}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid method", http.StatusBadRequest)
			return
		}

		vars := mux.Vars(r)
		token := vars["token"]

		req, err := requestFromToken(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Operation != storagepb.StoragePreSignUrlRequest_READ {
			http.Error(w, "invalid operation", http.StatusBadRequest)
			return
		}

		resp, err := storageService.Read(context.Background(), &storagepb.StorageReadRequest{
			BucketName: req.BucketName,
			Key:        req.Key,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

		_, err = w.Write(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	router.HandleFunc("/write/{token}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "invalid method", http.StatusBadRequest)
			return
		}

		vars := mux.Vars(r)
		token := vars["token"]

		req, err := requestFromToken(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Operation != storagepb.StoragePreSignUrlRequest_WRITE {
			http.Error(w, "invalid operation", http.StatusBadRequest)
			return
		}

		content, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = storageService.Write(context.Background(), &storagepb.StorageWriteRequest{
			BucketName: req.BucketName,
			Key:        req.Key,
			Body:       content,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

		_, err = w.Write([]byte("success"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	go func() {
		err := http.Serve(storageService.storageListener, router)
		if err != nil {
			logger.Errorf("Error serving storage listener: %s", err.Error())
		}
	}()

	return storageService, nil
}
