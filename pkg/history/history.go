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

package history

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"

	"github.com/nitrictech/cli/pkg/utils"
)

type HistoryRecords struct {
	ScheduleHistory []*HistoryRecord `json:"schedules"`
	TopicHistory    []*HistoryRecord `json:"topics"`
	ApiHistory      []*HistoryRecord `json:"apis"`
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

type EventRecord struct {
	WorkerKey string `json:"workerKey,omitempty"`
	TopicKey  string `json:"topicKey,omitempty"`
}

type EventHistoryItem struct {
	Event   *EventRecord `json:"event,omitempty"`
	Payload string       `json:"payload,omitempty"`
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

type History struct {
	ProjectDir string
}

func NewHistoryError(recordType RecordType, historyFile string) error {
	return fmt.Errorf("could not write %s history to the JSON file '%s' due to a formatting issue. Please check the file's formatting and ensure it follows the correct JSON structure, or reset the history by deleting the file", recordType, historyFile)
}

func (h *History) WriteHistoryRecord(recordType RecordType, historyRecord *HistoryRecord) error {
	historyFile, err := utils.NitricHistoryFile(h.ProjectDir, string(recordType))
	if err != nil {
		return err
	}

	existingRecords, err := h.ReadHistoryRecords(recordType)
	if err != nil {
		return NewHistoryError(recordType, historyFile)
	}

	existingRecords = append(existingRecords, historyRecord)

	data, err := json.Marshal(existingRecords)
	if err != nil {
		return NewHistoryError(recordType, historyFile)
	}

	err = os.WriteFile(historyFile, data, fs.ModePerm)
	if err != nil {
		return NewHistoryError(recordType, historyFile)
	}

	return nil
}

func (h *History) DeleteHistoryRecord(recordType RecordType) error {
	historyFile, err := utils.NitricHistoryFile(h.ProjectDir, string(recordType))
	if err != nil {
		return err
	}

	return os.Remove(historyFile)
}

func (h *History) ReadAllHistoryRecords() (*HistoryRecords, error) {
	schedules, err := h.ReadHistoryRecords(SCHEDULE)
	if err != nil {
		return nil, fmt.Errorf("error occurred reading schedule history: %w", err)
	}

	topics, err := h.ReadHistoryRecords(TOPIC)
	if err != nil {
		return nil, fmt.Errorf("error occurred reading topic history: %w", err)
	}

	apis, err := h.ReadHistoryRecords(API)
	if err != nil {
		return nil, fmt.Errorf("error occurred reading api history: %w", err)
	}

	return &HistoryRecords{
		ScheduleHistory: schedules,
		TopicHistory:    topics,
		ApiHistory:      apis,
	}, nil
}

func (h *History) ReadHistoryRecords(recordType RecordType) ([]*HistoryRecord, error) {
	historyFile, err := utils.NitricHistoryFile(h.ProjectDir, string(recordType))
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
