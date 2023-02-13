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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"go.etcd.io/bbolt"

	"github.com/nitrictech/nitric/core/pkg/plugins/document"
	"github.com/nitrictech/nitric/core/pkg/plugins/errors"
	"github.com/nitrictech/nitric/core/pkg/plugins/errors/codes"
	"github.com/nitrictech/nitric/core/pkg/utils"
)

const DEV_SUB_DIR_COLL = "./collections/"
const (
	skipTokenName    = "skip"
	idName           = "Id"
	partitionKeyName = "PartitionKey"
	sortKeyName      = "SortKey"
)

type BoltDocService struct {
	document.UnimplementedDocumentPlugin
	dbDir string
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

func (s *BoltDocService) Get(ctx context.Context, key *document.Key) (*document.Document, error) {
	newErr := errors.ErrorsWithScope(
		"BoltDocService.Get",
		map[string]interface{}{
			"key": key,
		},
	)

	if err := document.ValidateKey(key); err != nil {
		return nil, newErr(
			codes.InvalidArgument,
			"Invalid Key",
			err,
		)
	}

	db, err := s.createdDb(*key.Collection)
	if err != nil {
		return nil, newErr(
			codes.FailedPrecondition,
			"createDb error",
			err,
		)
	}

	defer db.Close()

	doc := createDoc(key)

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

	return toSdkDoc(key.Collection, doc), nil
}

func (s *BoltDocService) Set(ctx context.Context, key *document.Key, content map[string]interface{}) error {
	newErr := errors.ErrorsWithScope(
		"BoltDocService.Set",
		map[string]interface{}{
			"key": key,
		},
	)

	if err := document.ValidateKey(key); err != nil {
		return newErr(
			codes.InvalidArgument,
			"Invalid key",
			err,
		)
	}

	if content == nil {
		return newErr(
			codes.InvalidArgument,
			"Invalid content",
			nil,
		)
	}

	db, err := s.createdDb(*key.Collection)
	if err != nil {
		return newErr(
			codes.FailedPrecondition,
			"createDb error",
			err,
		)
	}

	defer db.Close()

	doc := createDoc(key)
	doc.Value = content

	if err := db.Save(&doc); err != nil {
		return newErr(
			codes.Internal,
			"Document save error",
			err,
		)
	}

	return nil
}

func (s *BoltDocService) Delete(ctx context.Context, key *document.Key) error {
	newErr := errors.ErrorsWithScope(
		"BoltDocService.Delete",
		map[string]interface{}{
			"key": key,
		},
	)

	if err := document.ValidateKey(key); err != nil {
		return newErr(
			codes.InvalidArgument,
			"Invalid key",
			err,
		)
	}

	db, err := s.createdDb(*key.Collection)
	if err != nil {
		return newErr(
			codes.FailedPrecondition,
			"createDb error",
			err,
		)
	}

	defer db.Close()

	doc := createDoc(key)

	err = db.DeleteStruct(&doc)
	if err != nil {
		return newErr(
			codes.Internal,
			"Deletion error",
			err,
		)
	}

	// Delete sub collection documents
	if key.Collection.Parent == nil {
		childDocs, err := fetchChildDocs(key, db)
		if err != nil {
			return newErr(
				codes.Internal,
				"Child Doc fetch error",
				err,
			)
		}

		for _, childDoc := range childDocs {
			err = db.DeleteStruct(&childDoc)
			if err != nil {
				return newErr(
					codes.Internal,
					"Child Doc deletion error",
					err,
				)
			}
		}
	}

	return nil
}

func (s *BoltDocService) query(collection *document.Collection, expressions []document.QueryExpression, limit int, pagingToken map[string]string, newErr errors.ErrorFactory) (*document.QueryResult, error) {
	if err := document.ValidateQueryCollection(collection); err != nil {
		return nil, newErr(
			codes.InvalidArgument,
			"Invalid Collection",
			err,
		)
	}

	if err := document.ValidateExpressions(expressions); err != nil {
		return nil, newErr(
			codes.InvalidArgument,
			"Invalid query expressions",
			err,
		)
	}

	db, err := s.createdDb(*collection)
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
		matchers = append(matchers, q.Lt(sortKeyName, document.GetEndRangeValue(collection.Name+"#")))
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
			expStr.WriteString(exp.Operand + " < '" + document.GetEndRangeValue(expValue) + "'")
		} else {
			if stringValue, ok := exp.Value.(string); ok {
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
	documents := make([]document.Document, 0)
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
		documents = append(documents, *sdkDoc)

		// Break if greater than fetch limit
		if limit > 0 && len(documents) == limit {
			break
		}
	}

	// Provide paging token to skip previous reads
	var resultPagingToken map[string]string
	if limit > 0 && len(documents) == limit {
		resultPagingToken = make(map[string]string)
		resultPagingToken[skipTokenName] = fmt.Sprintf("%v", pagingSkip+scanCount)
	}

	return &document.QueryResult{
		Documents:   documents,
		PagingToken: resultPagingToken,
	}, nil
}

func (s *BoltDocService) Query(ctx context.Context, collection *document.Collection, expressions []document.QueryExpression, limit int, pagingToken map[string]string) (*document.QueryResult, error) {
	newErr := errors.ErrorsWithScope(
		"BoltDocService.Query",
		map[string]interface{}{
			"collection": collection,
		},
	)

	return s.query(collection, expressions, limit, pagingToken, newErr)
}

func (s *BoltDocService) QueryStream(ctx context.Context, collection *document.Collection, expressions []document.QueryExpression, limit int) document.DocumentIterator {
	newErr := errors.ErrorsWithScope(
		"BoltDocService.QueryStream",
		map[string]interface{}{
			"collection": collection,
		},
	)

	tmpLimit := limit

	var (
		documents   []document.Document
		pagingToken map[string]string
	)

	// Initial fetch
	res, fetchErr := s.query(collection, expressions, limit, nil, newErr)

	if fetchErr != nil {
		// Return an error only iterator if the initial fetch failed
		return func() (*document.Document, error) {
			return nil, fetchErr
		}
	}

	documents = res.Documents
	pagingToken = res.PagingToken

	return func() (*document.Document, error) {
		// check the iteration state
		if tmpLimit == 0 && limit > 0 {
			// we've reached the limit of reading
			return nil, io.EOF
		} else if pagingToken != nil && len(documents) == 0 {
			// we've run out of documents and have more pages to read
			res, fetchErr = s.query(collection, expressions, tmpLimit, pagingToken, newErr)
			documents = res.Documents
			pagingToken = res.PagingToken
		} else if pagingToken == nil && len(documents) == 0 {
			// we're all out of documents and pages before hitting the limit
			return nil, io.EOF
		}

		// We received an error fetching the docs
		if fetchErr != nil {
			return nil, fetchErr
		}

		// pop the first element
		var doc document.Document
		doc, documents = documents[0], documents[1:]
		tmpLimit = tmpLimit - 1

		return &doc, nil
	}
}

// New - Create a new dev KV plugin
func NewBoltService() (*BoltDocService, error) {
	dbDir := utils.GetEnv("LOCAL_DB_DIR", utils.GetRelativeDevPath(DEV_SUB_DIR_COLL))

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

func (s *BoltDocService) createdDb(coll document.Collection) (*storm.DB, error) {
	for coll.Parent != nil {
		coll = *coll.Parent.Collection
	}

	dbPath := filepath.Join(s.dbDir, strings.ToLower(coll.Name)+".db")

	options := storm.BoltOptions(0o600, &bbolt.Options{Timeout: 1 * time.Second})

	db, err := storm.Open(dbPath, options)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func createDoc(key *document.Key) BoltDoc {
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
			Id:           parentKey.Id + document.SubcollectionDelimiter + key.Id,
			PartitionKey: parentKey.Id,
			SortKey:      key.Collection.Name + "#" + key.Id,
		}
	}
}

func toSdkDoc(col *document.Collection, doc BoltDoc) *document.Document {
	keys := strings.Split(doc.Id, document.SubcollectionDelimiter)

	// Translate the boltdb Id into a nitric document key Id
	var (
		id string
		c  *document.Collection
	)

	if len(keys) > 1 {
		// sub document
		id = keys[len(keys)-1]
		c = &document.Collection{
			Name: col.Name,
			Parent: &document.Key{
				Collection: col.Parent.Collection,
				Id:         keys[0],
			},
		}
	} else {
		id = doc.Id
		c = col
	}

	return &document.Document{
		Content: doc.Value,
		Key: &document.Key{
			Collection: c,
			Id:         id,
		},
	}
}

func fetchChildDocs(key *document.Key, db *storm.DB) ([]BoltDoc, error) {
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
