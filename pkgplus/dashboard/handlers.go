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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/nitrictech/cli/pkgplus/cloud/websockets"
	"github.com/nitrictech/cli/pkgplus/dashboard/history"
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

		if bucketName == "" && action != "list-buckets" {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			handleResponseWriter(w, []byte(`{"error": "Bucket is required"}`))

			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch action {
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

		// Remove "/api/call/" prefix from URL path
		path := strings.TrimPrefix(r.URL.Path, "/api/call/")

		// Build proxy request URL with query parameters
		targetURL := &url.URL{
			Scheme:   "http",
			Host:     callAddress,
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
		client := &http.Client{}

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

			err := d.history.DeleteHistoryRecord(history.RecordType(historyType))
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
