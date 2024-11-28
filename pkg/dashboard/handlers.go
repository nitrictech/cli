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

package dashboard

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkg/cloud/apis"
	"github.com/nitrictech/cli/pkg/cloud/batch"
	"github.com/nitrictech/cli/pkg/cloud/schedules"
	"github.com/nitrictech/cli/pkg/cloud/topics"
	"github.com/nitrictech/cli/pkg/cloud/websockets"
	base_http "github.com/nitrictech/nitric/cloud/common/runtime/gateway"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	secretspb "github.com/nitrictech/nitric/core/pkg/proto/secrets/v1"
	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
)

func (d *Dashboard) handleStorage() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		ctx := context.Background()
		bucketName := r.URL.Query().Get("bucket")
		action := r.URL.Query().Get("action")

		// Set the content type to JSON for all actions except read-file, to prevent it being set to text/plain
		if action != "read-file" {
			w.Header().Set("Content-Type", "application/json")
		}

		if bucketName == "" && action != "list-buckets" {
			w.WriteHeader(http.StatusBadRequest)
			handleResponseWriter(w, []byte(`{"error": "Bucket is required"}`))

			return
		}

		switch action {
		case "read-file":
			fileKey := r.URL.Query().Get("fileKey")
			if fileKey == "" {
				w.WriteHeader(http.StatusBadRequest)
				handleResponseWriter(w, []byte(`{"error": "fileKey is required for delete-file action"}`))

				return
			}

			resp, err := d.storageService.Read(ctx, &storagepb.StorageReadRequest{
				BucketName: bucketName,
				Key:        fileKey,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			handleResponseWriter(w, resp.Body)

			return
		case "list-files":
			fileList, err := d.storageService.ListBlobs(ctx, &storagepb.StorageListBlobsRequest{
				BucketName: bucketName,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			jsonResponse, err := json.Marshal(fileList.GetBlobs())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			handleResponseWriter(w, jsonResponse)

			return
		case "write-file":
			fileKey := r.URL.Query().Get("fileKey")
			if fileKey == "" {
				w.WriteHeader(http.StatusBadRequest)
				handleResponseWriter(w, []byte(`{"error": "fileKey is required for delete-file action"}`))

				return
			}

			// Read the contents of the file
			contents, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				handleResponseWriter(w, []byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))

				return
			}

			_, err = d.storageService.Write(ctx, &storagepb.StorageWriteRequest{
				BucketName: bucketName,
				Key:        fileKey,
				Body:       contents,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			handleResponseWriter(w, []byte(`{"success": true}`))

			return
		case "delete-file":
			fileKey := r.URL.Query().Get("fileKey")
			if fileKey == "" {
				w.WriteHeader(http.StatusBadRequest)
				handleResponseWriter(w, []byte(`{"error": "fileKey is required for delete-file action"}`))

				return
			}

			_, err := d.storageService.Delete(ctx, &storagepb.StorageDeleteRequest{
				BucketName: bucketName,
				Key:        fileKey,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			handleResponseWriter(w, []byte(`{"success": true}`))

			return
		default:
			handleResponseWriter(w, []byte(`{"error": "Invalid action"}`))
		}
	}
}

func (d *Dashboard) createCallProxyHttpHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORs headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// find call callAddress
		callAddress := r.Header.Get("X-Nitric-Local-Call-Address")
		// Assume http if no scheme is provided, since that was the previous default behavior
		if !(regexp.MustCompile(`^.*://`).MatchString(callAddress)) {
			callAddress = "http://" + callAddress
		}

		// Remove "/api/call/" prefix from URL path
		path := strings.TrimPrefix(r.URL.Path, "/api/call/")

		// Parse the callAddress as a URL
		callUrl, err := url.ParseRequestURI(callAddress)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Build proxy request URL with query parameters
		targetURL := &url.URL{
			Scheme:   callUrl.Scheme,
			Host:     callUrl.Host,
			Path:     path,
			RawQuery: r.URL.RawQuery,
		}

		// Create a new request object
		req, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Copy the headers from the original request to the new request
		for key, value := range r.Header {
			req.Header.Set(key, value[0])
		}

		// Send the new request and handle the response
		client := &http.Client{
			// skip tls verification, since local services can use self-signed certs
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()

		// Copy the headers from the response to the response writer
		for key, value := range resp.Header {
			w.Header().Set(key, value[0])
		}

		// Copy the status code from the response to the response writer
		w.WriteHeader(resp.StatusCode)

		// Copy the response body to the response writer
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (d *Dashboard) createSqlQueryHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORs headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Parse the SQL query from the request body
		var requestBody struct {
			Query            string `json:"query"`
			ConnectionString string `json:"connectionString"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Execute the SQL query
		results, err := d.databaseService.Query(context.Background(), requestBody.ConnectionString, requestBody.Query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Write the results to the response
		jsonResponse, err := json.Marshal(results)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err = w.Write(jsonResponse)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (d *Dashboard) createApplySqlMigrationsHandler(fs afero.Fs, useBuilder bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORs headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Parse the SQL migrations from the request body
		var requestBody struct {
			DatabaseName string `json:"databaseName"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if requestBody.DatabaseName == "" {
			http.Error(w, "missing databaseName param", http.StatusBadRequest)
			return
		}

		// Apply the SQL migrations
		databasesToMigrate := map[string]*resourcespb.SqlDatabaseResource{}

		dbState := d.databaseService.GetState()

		if _, ok := dbState[requestBody.DatabaseName]; !ok {
			http.Error(w, "database not found", http.StatusBadRequest)
			return
		}

		databasesToMigrate[requestBody.DatabaseName] = dbState[requestBody.DatabaseName].ResourceRegister.Resource

		err = d.databaseService.BuildAndRunMigrations(fs, databasesToMigrate, useBuilder)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (d *Dashboard) createSecretsHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		secretName := r.URL.Query().Get("secret")
		action := r.URL.Query().Get("action")
		version := r.URL.Query().Get("version")
		latest := r.URL.Query().Get("latest")

		if secretName == "" {
			http.Error(w, "missing secret param", http.StatusBadRequest)
			return
		}

		switch action {
		case "list-versions":
			secretVersions, err := d.secretService.List(context.Background(), secretName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			jsonResponse, err := json.Marshal(secretVersions)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			_, err = w.Write(jsonResponse)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case "add-secret-version":
			// get data from body
			var requestBody struct {
				Value string `json:"value"`
			}

			err := json.NewDecoder(r.Body).Decode(&requestBody)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if requestBody.Value == "" {
				http.Error(w, "missing value param", http.StatusBadRequest)
				return
			}

			_, err = d.secretService.Put(context.Background(), &secretspb.SecretPutRequest{
				Secret: &secretspb.Secret{
					Name: secretName,
				},
				Value: []byte(requestBody.Value),
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
		case "delete-secret":
			if version == "" {
				http.Error(w, "missing version param", http.StatusBadRequest)
				return
			}

			err := d.secretService.Delete(context.Background(), secretName, version, latest == "true")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "invalid action", http.StatusBadRequest)
			return
		}
	}
}

func (d *Dashboard) createHistoryHttpHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == "DELETE" {
			historyType := r.URL.Query().Get("type")

			err := d.DeleteHistoryRecord(RecordType(historyType))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)

		err := d.sendHistoryUpdate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (d *Dashboard) handleWebsocketMessagesClear() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "DELETE" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		socketName := r.URL.Query().Get("socket")

		if socketName == "" {
			http.Error(w, "missing socket param", http.StatusBadRequest)
			return
		}

		if d.websocketsInfo[socketName] == nil {
			http.Error(w, "socket not found", http.StatusNotFound)
			return
		}

		d.websocketsInfo[socketName].Messages = []websockets.WebsocketMessage{}

		w.WriteHeader(http.StatusOK)

		err := d.sendWebsocketsUpdate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (d *Dashboard) handleApiHistory(state apis.ApiRequestState) {
	var queryParams []Param

	state.ReqCtx.QueryArgs().VisitAll(func(key []byte, val []byte) {
		queryParams = append(queryParams, Param{
			Key:   string(key),
			Value: string(val),
		})
	})

	err := d.writeHistoryRecord(&HistoryEvent[any]{
		Time:       time.Now().UnixMilli(),
		RecordType: API,
		Event: ApiHistoryItem{
			Api: state.Api,
			Request: &RequestHistory{
				Method:      string(state.ReqCtx.Request.Header.Method()),
				Path:        string(state.ReqCtx.URI().PathOriginal()),
				QueryParams: queryParams,
				Headers:     base_http.HttpHeadersToMap(&state.ReqCtx.Request.Header),
				Body:        state.ReqCtx.Request.Body(),
				PathParams:  []Param{},
			},
			Response: &ResponseHistory{
				Headers: lo.MapEntries(state.HttpResp.Headers, func(k string, v *apispb.HeaderValue) (string, []string) {
					return k, v.Value
				}),
				Time:   time.Since(state.ReqCtx.ConnTime()).Milliseconds(),
				Status: state.HttpResp.GetStatus(),
				Data:   state.HttpResp.GetBody(),
				Size:   len(state.HttpResp.GetBody()),
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Dashboard) handleWebsocketEvents(action websockets.WebsocketAction[websockets.EventItem]) {
	if d.websocketsInfo[action.Name] == nil {
		d.websocketsInfo[action.Name] = &websockets.WebsocketInfo{}
	}

	switch e := action.Event.(type) {
	case websockets.WebsocketInfo:
		d.websocketsInfo[action.Name].ConnectionCount = e.ConnectionCount
	case websockets.WebsocketMessage:
		d.websocketsInfo[action.Name].Messages = append([]websockets.WebsocketMessage{e}, d.websocketsInfo[action.Name].Messages...)
	}

	err := d.sendWebsocketsUpdate()
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Dashboard) handleTopicsHistory(action topics.ActionState) {
	err := d.writeHistoryRecord(&HistoryEvent[any]{
		Time:       time.Now().UnixMilli(),
		RecordType: TOPIC,
		Event: TopicHistoryItem{
			Name:    action.TopicName,
			Payload: action.Payload,
			Success: action.Success,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Dashboard) handleSchedulesHistory(action schedules.ActionState) {
	err := d.writeHistoryRecord(&HistoryEvent[any]{
		Time:       time.Now().UnixMilli(),
		RecordType: SCHEDULE,
		Event: ScheduleHistoryItem{
			Name:    action.ScheduleName,
			Success: action.Success,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Dashboard) handleBatchJobsHistory(action batch.ActionState) {
	err := d.writeHistoryRecord(&HistoryEvent[any]{
		Time:       time.Now().UnixMilli(),
		RecordType: BATCHJOBS,
		Event: BatchHistoryItem{
			Name:    action.JobName,
			Payload: action.Payload,
			Success: action.Success,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
