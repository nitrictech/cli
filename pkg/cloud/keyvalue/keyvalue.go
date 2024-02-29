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

package keyvalue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"go.etcd.io/bbolt"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nitrictech/cli/pkg/cloud/env"
	grpc_errors "github.com/nitrictech/nitric/core/pkg/grpc/errors"
	kvstorepb "github.com/nitrictech/nitric/core/pkg/proto/kvstore/v1"
)

const DEV_SUB_DIR_COLL = "./kv/"
const (
	skipTokenName    = "skip"
	idName           = "Id"
	partitionKeyName = "PartitionKey"
	sortKeyName      = "SortKey"
)

type BoltDocService struct {
	dbDir string
}

var _ kvstorepb.KvStoreServer = (*BoltDocService)(nil)

// GetEndRangeValue - Get end range value to implement "startsWith" expression operator using where clause.
// For example with sdk.Expression("pk", "startsWith", "Customer#") this translates to:
// WHERE pk >= {startRangeValue} AND pk < {endRangeValue}
// WHERE pk >= "Customer#" AND pk < "Customer!"
func GetEndRangeValue(value string) string {
	strFrontCode := value[:len(value)-1]

	strEndCode := value[len(value)-1:]

	return strFrontCode + string(strEndCode[0]+1)
}

type BoltDoc struct {
	Id           string `storm:"id"`
	PartitionKey string `storm:"index"`
	SortKey      string `storm:"index"`
	Value        map[string]interface{}
}

func (d BoltDoc) String() string {
	return fmt.Sprintf("BoltDoc{Id: %v PartitionKey: %v SortKey: %v Value: %v}\n", d.Id, d.PartitionKey, d.SortKey, d.Value)
}

func (s *BoltDocService) GetValue(ctx context.Context, req *kvstorepb.KvStoreGetValueRequest) (*kvstorepb.KvStoreGetValueResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Get")

	db, err := s.getLocalKVDB(req.Ref.Store)
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"createDb error",
			err,
		)
	}

	defer db.Close()

	doc := createDoc(req.Ref)

	err = db.One(idName, doc.Id, &doc)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, newErr(
				codes.NotFound,
				"document not found",
				err,
			)
		}

		return nil, newErr(
			codes.Internal,
			"DB Fetch error",
			err,
		)
	}

	sdkDoc, err := toSdkDoc(req.Ref, doc)
	if err != nil {
		return nil, newErr(
			codes.Internal,
			"toSdkDoc error",
			err,
		)
	}

	return &kvstorepb.KvStoreGetValueResponse{
		Value: sdkDoc,
	}, nil
}

func (s *BoltDocService) SetValue(ctx context.Context, req *kvstorepb.KvStoreSetValueRequest) (*kvstorepb.KvStoreSetValueResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Set")

	db, err := s.getLocalKVDB(req.Ref.Store)
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"createDb error",
			err,
		)
	}

	defer db.Close()

	doc := createDoc(req.Ref)
	doc.Value = req.Content.AsMap()

	if err := db.Save(&doc); err != nil {
		return nil, newErr(
			codes.Internal,
			"Document save error",
			err,
		)
	}

	return &kvstorepb.KvStoreSetValueResponse{}, nil
}

func (s *BoltDocService) DeleteKey(ctx context.Context, req *kvstorepb.KvStoreDeleteKeyRequest) (*kvstorepb.KvStoreDeleteKeyResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Delete")

	key := req.Ref

	db, err := s.getLocalKVDB(key.Store)
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"createDb error",
			err,
		)
	}

	defer db.Close()

	doc := createDoc(key)

	err = db.DeleteStruct(&doc)
	if err != nil {
		return nil, newErr(
			codes.Internal,
			"Deletion error",
			err,
		)
	}

	return &kvstorepb.KvStoreDeleteKeyResponse{}, nil
}

// New - Create a new dev KV plugin
func NewBoltService() (*BoltDocService, error) {
	dbDir := env.LOCAL_DB_DIR.String()

	// Check whether file exists
	_, err := os.Stat(dbDir)
	if os.IsNotExist(err) {
		// Make directory if not present
		err := os.MkdirAll(dbDir, 0o777)
		if err != nil {
			return nil, err
		}
	}

	return &BoltDocService{dbDir: dbDir}, nil
}

func (s *BoltDocService) ScanKeys(req *kvstorepb.KvStoreScanKeysRequest, stream kvstorepb.KvStore_ScanKeysServer) error {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Keys")
	storeName := req.GetStore().GetName()

	if storeName == "" {
		return newErr(
			codes.InvalidArgument,
			"store name is required",
			nil,
		)
	}

	db, err := s.getLocalKVDB(storeName)
	if err != nil {
		return newErr(
			codes.Internal,
			"failed to retrieve key/value store",
			err,
		)
	}

	defer db.Close()

	prefixPattern := "^" + regexp.QuoteMeta(req.GetPrefix())

	var docs []BoltDoc

	err = db.Select(q.Re(idName, prefixPattern)).Find(&docs)
	if err != nil {
		// not found isn't an error, just close the stream and return no results
		if errors.Is(err, storm.ErrNotFound) {
			return nil
		}

		return newErr(
			codes.Internal,
			"failed query key/value store",
			err,
		)
	}

	for _, doc := range docs {
		if err := stream.Send(&kvstorepb.KvStoreScanKeysResponse{
			Key: doc.Id,
		}); err != nil {
			return newErr(
				codes.Internal,
				"failed to send response",
				err,
			)
		}
	}

	return nil
}

func (s *BoltDocService) getLocalKVDB(storeName string) (*storm.DB, error) {
	dbPath := filepath.Join(s.dbDir, strings.ToLower(storeName)+".db")

	options := storm.BoltOptions(0o600, &bbolt.Options{Timeout: 1 * time.Second})

	db, err := storm.Open(dbPath, options)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func createDoc(key *kvstorepb.ValueRef) BoltDoc {
	return BoltDoc{
		Id:           key.Key,
		PartitionKey: key.Key,
		SortKey:      key.Store,
	}
}

func toSdkDoc(ref *kvstorepb.ValueRef, doc BoltDoc) (*kvstorepb.Value, error) {
	content, err := structpb.NewStruct(doc.Value)
	if err != nil {
		// FIXME: Handle error...
		return nil, err
	}

	return &kvstorepb.Value{
		Ref:     ref,
		Content: content,
	}, nil
}
