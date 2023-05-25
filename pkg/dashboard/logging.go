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
	"fmt"
	"io/fs"
	"os"

	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/utils"
)

type History struct {
	ScheduleHistory []*HistoryRecord `json:"scheduleHistory,omitempty"`
	TopicHistory    []*HistoryRecord `json:"topicHistory,omitempty"`
	ApiHistory      []*HistoryRecord `json:"apiHistory,omitempty"`
}

type RecordType string

const (
	API      RecordType = "apis"
	TOPIC    RecordType = "topics"
	SCHEDULE RecordType = "schedules"
)

type TriggerType string

type HistoryRecord struct {
	Time    int64 `json:"time,omitempty"`
	Success bool  `json:"success,omitempty"`
	EventHistoryItem
	ApiHistoryItem
}

type EventHistoryItem struct {
	Event   *codeconfig.TopicResult `json:"event,omitempty"`
	Payload string                  `json:"payload,omitempty"`
}

type ApiHistoryItem struct {
	Api      string           `json:"api"`
	Request  *RequestHistory  `json:"request"`
	Response *ResponseHistory `json:"response"`
}

type Param struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// Only log what is required from req/resp to avoid massive log files
type RequestHistory struct {
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	QueryParams []Param             `json:"queryParams"`
	PathParams  []Param             `json:"pathParams"`
	Body        []byte              `json:"body"`
	Headers     map[string][]string `json:"headers"`
}

type ResponseHistory struct {
	Data    interface{}         `json:"data"`
	Status  int32               `json:"status"`
	Size    int                 `json:"size"`
	Time    int64               `json:"time"`
	Headers map[string][]string `json:"headers"`
}

func WriteHistoryRecord(stackName string, recordType RecordType, historyRecord *HistoryRecord) error {
	historyFile, err := utils.NitricHistoryFile(stackName, string(recordType))
	if err != nil {
		return err
	}

	existingRecords, err := ReadHistoryRecords(stackName, recordType)
	if err != nil {
		return err
	}

	existingRecords = append(existingRecords, historyRecord)

	data, err := json.Marshal(existingRecords)
	if err != nil {
		return err
	}

	return os.WriteFile(historyFile, data, fs.ModePerm)
}

func DeleteHistoryRecord(stackName string, recordType RecordType) error {
	historyFile, err := utils.NitricHistoryFile(stackName, string(recordType))
	if err != nil {
		return err
	}

	return os.Remove(historyFile)
}

func ReadAllHistoryRecords(stackName string) (*History, error) {
	schedules, err := ReadHistoryRecords(stackName, SCHEDULE)
	if err != nil {
		return nil, fmt.Errorf("error occurred reading schedule history: %w", err)
	}

	topics, err := ReadHistoryRecords(stackName, TOPIC)
	if err != nil {
		return nil, fmt.Errorf("error occurred reading topic history: %w", err)
	}

	apis, err := ReadHistoryRecords(stackName, API)
	if err != nil {
		return nil, fmt.Errorf("error occurred reading api history: %w", err)
	}

	return &History{
		ScheduleHistory: schedules,
		TopicHistory:    topics,
		ApiHistory:      apis,
	}, nil
}

func ReadHistoryRecords(stackName string, recordType RecordType) ([]*HistoryRecord, error) {
	historyFile, err := utils.NitricHistoryFile(stackName, string(recordType))
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(historyFile)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return []*HistoryRecord{}, nil
	}

	var history []*HistoryRecord

	err = json.Unmarshal(data, &history)
	if err != nil {
		return nil, err
	}

	return history, nil
}
