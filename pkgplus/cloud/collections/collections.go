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

package collections

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"go.etcd.io/bbolt"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	"errors"

	"github.com/nitrictech/cli/pkgplus/cloud/env"
	grpc_errors "github.com/nitrictech/nitric/core/pkg/grpc/errors"
	documentspb "github.com/nitrictech/nitric/core/pkg/proto/documents/v1"
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

var _ documentspb.DocumentsServer = (*BoltDocService)(nil)

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

func (s *BoltDocService) Get(ctx context.Context, req *documentspb.DocumentGetRequest) (*documentspb.DocumentGetResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Get")

	db, err := s.getLocalCollectionDB(req.Key.Collection)
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"createDb error",
			err,
		)
	}

	defer db.Close()

	doc := createDoc(req.Key)

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

	return &documentspb.DocumentGetResponse{
		Document: toSdkDoc(req.Key.Collection, doc),
	}, nil
}

func (s *BoltDocService) Set(ctx context.Context, req *documentspb.DocumentSetRequest) (*documentspb.DocumentSetResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Set")

	db, err := s.getLocalCollectionDB(req.Key.Collection)
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"createDb error",
			err,
		)
	}

	defer db.Close()

	doc := createDoc(req.Key)
	doc.Value = req.Content.AsMap()

	if err := db.Save(&doc); err != nil {
		return nil, newErr(
			codes.Internal,
			"Document save error",
			err,
		)
	}

	return &documentspb.DocumentSetResponse{}, nil
}

func (s *BoltDocService) Delete(ctx context.Context, req *documentspb.DocumentDeleteRequest) (*documentspb.DocumentDeleteResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Delete")

	key := req.Key

	db, err := s.getLocalCollectionDB(key.Collection)
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"createDb error",
			err,
		)
	}

	defer db.Close()

	doc := createDoc(req.Key)

	err = db.DeleteStruct(&doc)
	if err != nil {
		return nil, newErr(
			codes.Internal,
			"Deletion error",
			err,
		)
	}

	// Delete sub collection documents
	if req.Key.Collection.Parent == nil {
		childDocs, err := fetchChildDocs(req.Key, db)
		if err != nil {
			return nil, newErr(
				codes.Internal,
				"Child Doc fetch error",
				err,
			)
		}

		for _, childDoc := range childDocs {
			err = db.DeleteStruct(&childDoc)
			if err != nil {
				return nil, newErr(
					codes.Internal,
					"Child Doc deletion error",
					err,
				)
			}
		}
	}

	return &documentspb.DocumentDeleteResponse{}, nil
}

func (s *BoltDocService) query(collection *documentspb.Collection, expressions []*documentspb.Expression, limit int32, pagingToken map[string]string, newErr grpc_errors.ScopedErrorFactory) (*documentspb.DocumentQueryResponse, error) {

	db, err := s.getLocalCollectionDB(collection)
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"createDb error",
			err,
		)
	}
	defer db.Close()

	// Build up chain of expression matchers
	matchers := []q.Matcher{}

	// Apply collection/sub-collection filters
	parentKey := collection.Parent

	if parentKey == nil {
		matchers = append(matchers, q.Eq(sortKeyName, collection.Name+"#"))
	} else {
		if parentKey.Id != "" {
			matchers = append(matchers, q.Eq(partitionKeyName, parentKey.Id))
		}
		matchers = append(matchers, q.Gte(sortKeyName, collection.Name+"#"))
		matchers = append(matchers, q.Lt(sortKeyName, GetEndRangeValue(collection.Name+"#")))
	}

	// Create query object
	matcher := q.And(matchers[:]...)
	query := db.Select(matcher)

	pagingSkip := 0

	// If fetch limit configured skip past previous reads
	if limit > 0 && len(pagingToken) > 0 {
		if val, found := pagingToken[skipTokenName]; found {
			pagingSkip, err = strconv.Atoi(val)
			if err != nil {
				return nil, newErr(
					codes.InvalidArgument,
					"Invalid paging token",
					err,
				)
			}

			query = query.Skip(pagingSkip)
		}
	}

	// Execute query
	var docs []BoltDoc

	err = query.Find(&docs)
	if err != nil {
		fmt.Println("query.Find: ", err)
	}

	// Create values map filter expression, for example : country == 'US' && age < '12'
	expStr := strings.Builder{}

	for _, exp := range expressions {
		// TODO: test typing capabilities of library and rewrite expressions based on value type
		expValue := fmt.Sprintf("%v", exp.Value)

		if expStr.Len() > 0 {
			expStr.WriteString(" && ")
		}

		if exp.Operator == "startsWith" {
			expStr.WriteString(exp.Operand + " >= '" + expValue + "' && ")
			expStr.WriteString(exp.Operand + " < '" + GetEndRangeValue(expValue) + "'")
		} else {
			if stringValue := exp.GetValue().GetStringValue(); stringValue != "" {
				expValue = fmt.Sprintf("'%s'", stringValue)
			}

			expStr.WriteString(exp.Operand + " " + exp.Operator + " " + expValue)
		}
	}

	var filterExp *govaluate.EvaluableExpression
	if expStr.Len() > 0 {
		filterExp, err = govaluate.NewEvaluableExpression(expStr.String())
		if err != nil {
			return nil, newErr(
				codes.InvalidArgument,
				fmt.Sprintf("Unable to create filter expressions from: %s", expStr.String()),
				err,
			)
		}
	}

	// Process query results, applying value filter expressions and fetch limit
	documents := make([]*documentspb.Document, 0)
	scanCount := 0

	for _, doc := range docs {
		scanCount += 1

		if filterExp != nil {
			include, err := filterExp.Evaluate(doc.Value)
			if err != nil || !(include.(bool)) {
				// TODO: determine if skipping failed evaluations is always appropriate.
				// 	errors are usually a datatype mismatch or a missing key/prop on the doc, which is essentially a failed match.
				// Treat a failed or false eval as a mismatch
				continue
			}
		}

		sdkDoc := toSdkDoc(collection, doc)
		documents = append(documents, sdkDoc)

		// Break if greater than fetch limit
		if limit > 0 && len(documents) == int(limit) {
			break
		}
	}

	// Provide paging token to skip previous reads
	var resultPagingToken map[string]string
	if limit > 0 && len(documents) == int(limit) {
		resultPagingToken = make(map[string]string)
		resultPagingToken[skipTokenName] = fmt.Sprintf("%v", pagingSkip+scanCount)
	}

	return &documentspb.DocumentQueryResponse{
		Documents:   documents,
		PagingToken: resultPagingToken,
	}, nil
}

func (s *BoltDocService) Query(ctx context.Context, req *documentspb.DocumentQueryRequest) (*documentspb.DocumentQueryResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.Query")

	return s.query(req.Collection, req.Expressions, req.Limit, req.PagingToken, newErr)
}

func (s *BoltDocService) QueryStream(req *documentspb.DocumentQueryStreamRequest, stream documentspb.Documents_QueryStreamServer) error {
	newErr := grpc_errors.ErrorsWithScope("BoltDocService.QueryStream")

	limitCountdown := req.Limit

	var (
		documents   []*documentspb.Document
		pagingToken map[string]string
	)

	// Initial fetch
	res, fetchErr := s.query(req.Collection, req.Expressions, limitCountdown, nil, newErr)

	if fetchErr != nil {
		// TODO: determine if this is the correct error code to return
		return newErr(codes.Unknown, "failed to initiate document query", fetchErr)
	}

	documents = res.Documents
	pagingToken = res.PagingToken

	for {
		// check the iteration state
		if limitCountdown == 0 && req.Limit > 0 {
			// we've reached the limit of reading
			return nil
		}

		if pagingToken == nil && len(documents) == 0 {
			// no more documents to return, regardless of the limit
			return nil
		}

		if pagingToken != nil && len(documents) == 0 {
			// we've run out of documents in our buffer but still have more to read (cursor is still set)
			res, fetchErr = s.query(req.Collection, req.Expressions, limitCountdown, pagingToken, newErr)
			// We received an error fetching the docs
			if fetchErr != nil {
				return fetchErr
			}

			documents = res.Documents
			pagingToken = res.PagingToken
		}

		// pop the first element
		var doc *documentspb.Document
		doc, documents = documents[0], documents[1:]
		limitCountdown = limitCountdown - 1

		stream.Send(&documentspb.DocumentQueryStreamResponse{
			Document: doc,
		})
	}
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

func (s *BoltDocService) getLocalCollectionDB(coll *documentspb.Collection) (*storm.DB, error) {
	for coll.Parent != nil {
		coll = coll.Parent.Collection
	}

	dbPath := filepath.Join(s.dbDir, strings.ToLower(coll.Name)+".db")

	options := storm.BoltOptions(0o600, &bbolt.Options{Timeout: 1 * time.Second})

	db, err := storm.Open(dbPath, options)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func createDoc(key *documentspb.Key) BoltDoc {
	parentKey := key.Collection.Parent

	// Top Level Collection
	if parentKey == nil {
		return BoltDoc{
			Id:           key.Id,
			PartitionKey: key.Id,
			SortKey:      key.Collection.Name + "#",
		}
	} else {
		return BoltDoc{
			Id:           parentKey.Id + subcollectionDelimiter + key.Id,
			PartitionKey: parentKey.Id,
			SortKey:      key.Collection.Name + "#" + key.Id,
		}
	}
}

func toSdkDoc(col *documentspb.Collection, doc BoltDoc) *documentspb.Document {
	keys := strings.Split(doc.Id, subcollectionDelimiter)

	// Translate the boltdb Id into a nitric document key Id
	var (
		id string
		c  *documentspb.Collection
	)

	if len(keys) > 1 {
		// sub document
		id = keys[len(keys)-1]
		c = &documentspb.Collection{
			Name: col.Name,
			Parent: &documentspb.Key{
				Collection: col.Parent.Collection,
				Id:         keys[0],
			},
		}
	} else {
		id = doc.Id
		c = col
	}

	content, err := structpb.NewStruct(doc.Value)
	if err != nil {
		// FIXME: Handle error...
		panic(err)
	}

	return &documentspb.Document{
		Content: content,
		Key: &documentspb.Key{
			Collection: c,
			Id:         id,
		},
	}
}

func fetchChildDocs(key *documentspb.Key, db *storm.DB) ([]BoltDoc, error) {
	var childDocs []BoltDoc

	err := db.Find(partitionKeyName, key.Id, &childDocs)
	if err != nil {
		if err.Error() == "not found" {
			return childDocs, nil
		} else {
			return nil, err
		}
	}

	return childDocs, nil
}
