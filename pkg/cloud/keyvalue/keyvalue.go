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
	"strings"
	"time"

	"github.com/asdine/storm"
	"go.etcd.io/bbolt"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nitrictech/cli/pkg/cloud/env"
	grpc_errors "github.com/nitrictech/nitric/core/pkg/grpc/errors"
	keyvaluepb "github.com/nitrictech/nitric/core/pkg/proto/keyvalue/v1"
)

const DEV_SUB_DIR_COLL = "./collections/"
const (
	skipTokenName    = "skip"
	idName           = "Id"
	partitionKeyName = "PartitionKey"
	sortKeyName      = "SortKey"
)

const subcollectionDelimiter = "+"

type BoltDocService struct {
	dbDir string
}

var _ keyvaluepb.KeyValueServer = (*BoltDocService)(nil)

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

func (s *BoltDocService) Get(ctx context.Context, req *keyvaluepb.KeyValueGetRequest) (*keyvaluepb.KeyValueGetResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Get")

	db, err := s.getLocalCollectionDB(req.Ref)
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

	return &keyvaluepb.KeyValueGetResponse{
		Value: sdkDoc,
	}, nil
}

func (s *BoltDocService) Set(ctx context.Context, req *keyvaluepb.KeyValueSetRequest) (*keyvaluepb.KeyValueSetResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Set")

	db, err := s.getLocalCollectionDB(req.Ref)
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

	return &keyvaluepb.KeyValueSetResponse{}, nil
}

func (s *BoltDocService) Delete(ctx context.Context, req *keyvaluepb.KeyValueDeleteRequest) (*keyvaluepb.KeyValueDeleteResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Delete")

	key := req.Ref

	db, err := s.getLocalCollectionDB(key)
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

	return &keyvaluepb.KeyValueDeleteResponse{}, nil
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

func (s *BoltDocService) getLocalCollectionDB(coll *keyvaluepb.ValueRef) (*storm.DB, error) {
	dbPath := filepath.Join(s.dbDir, strings.ToLower(coll.Store)+".db")

	options := storm.BoltOptions(0o600, &bbolt.Options{Timeout: 1 * time.Second})

	db, err := storm.Open(dbPath, options)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func createDoc(key *keyvaluepb.ValueRef) BoltDoc {
	return BoltDoc{
		Id:           key.Key,
		PartitionKey: key.Key,
		SortKey:      key.Store,
	}
}

func toSdkDoc(ref *keyvaluepb.ValueRef, doc BoltDoc) (*keyvaluepb.Value, error) {
	content, err := structpb.NewStruct(doc.Value)
	if err != nil {
		// FIXME: Handle error...
		return nil, err
	}

	return &keyvaluepb.Value{
		Ref:     ref,
		Content: content,
	}, nil
}
