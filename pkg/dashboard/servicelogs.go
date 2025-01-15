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
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/system"
)

type TimelineFilter string

const (
	PastHour     TimelineFilter = "pastHour"
	PastHalfHour TimelineFilter = "pastHalfHour"
	PastWeek     TimelineFilter = "pastWeek"
)

func (d *Dashboard) createServiceLogsHandler(project *project.Project) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == "GET" {
			logs, err := system.ReadLogs()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			originFilter := r.URL.Query().Get("origin")
			levelFilter := r.URL.Query().Get("level")
			searchFilter := r.URL.Query().Get("search")
			timelineFilter := r.URL.Query().Get("timeline")

			filteredLogs := filterLogs(logs, originFilter, levelFilter, searchFilter, timelineFilter)

			// Send logs as JSON response
			w.Header().Set("Content-Type", "application/json")

			if err := json.NewEncoder(w).Encode(filteredLogs); err != nil {
				http.Error(w, "Failed to encode logs: "+err.Error(), http.StatusInternalServerError)
			}

			return
		}

		if r.Method == "DELETE" {
			err := system.PurgeLogs()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
		}
	}
}

// Helper function to filter logs using lo.Filter
func filterLogs(logs []system.LogEntry, originFilter, levelFilter, searchFilter, timelineFilter string) []system.LogEntry {
	var origins, levels []string

	if originFilter != "" {
		origins = strings.Split(originFilter, ",")
	}

	if levelFilter != "" {
		levels = strings.Split(levelFilter, ",")
	}

	// Parse startDate and endDate based on the timelineFilter
	var startDate, endDate time.Time

	if timelineFilter != "" {
		currentTime := time.Now()

		switch TimelineFilter(timelineFilter) {
		case PastHour:
			startDate = currentTime.Add(-1 * time.Hour)
		case PastHalfHour:
			startDate = currentTime.Add(-30 * time.Minute)
		case PastWeek:
			startDate = currentTime.Add(-7 * 24 * time.Hour)
		}

		endDate = currentTime
	}

	filteredLogs := lo.Filter(logs, func(log system.LogEntry, _ int) bool {
		matchesOrigin := len(origins) == 0 || lo.Contains(origins, log.Origin)
		matchesLevel := len(levels) == 0 || lo.Contains(levels, log.Level.String())
		matchesSearch := searchFilter == "" || strings.Contains(strings.ToLower(log.Message), strings.ToLower(searchFilter))

		// Check if the log's timestamp is within the timeline range
		matchesDate := true

		if !startDate.IsZero() && !endDate.IsZero() {
			matchesDate = log.Timestamp.Local().After(startDate) && log.Timestamp.Local().Before(endDate)
		}

		return matchesOrigin && matchesLevel && matchesSearch && matchesDate
	})

	// Reverse the order to show newest logs first
	sort.Slice(filteredLogs, func(i, j int) bool {
		return filteredLogs[i].Timestamp.After(filteredLogs[j].Timestamp)
	})

	return filteredLogs
}
