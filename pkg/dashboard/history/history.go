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

	"github.com/nitrictech/cli/pkg/eventbus"
	"github.com/nitrictech/cli/pkg/utils"
)

const AddRecordTopic = "history:addrecord"

type HistoryEvents struct {
	ScheduleHistory []*HistoryEvent[any]            `json:"schedules"`
	TopicHistory    []*HistoryEvent[TopicEvent]     `json:"topics"`
	ApiHistory      []*HistoryEvent[ApiHistoryItem] `json:"apis"`
}

type RecordType string

const (
	API      RecordType = "apis"
	TOPIC    RecordType = "topics"
	SCHEDULE RecordType = "schedules"
)

type TriggerType string

type HistoryEvent[Event any] struct {
	Time int64 `json:"time,omitempty"`
	// Success    bool  `json:"success,omitempty"`
	Event      Event
	RecordType RecordType `json:"-"`
}

type TopicEvent struct {
	Id      string `json:"Id"`
	Topic   string `json:"topic"`
	Publish *TopicPublishEvent
	Result  *TopicSubscriberResultEvent
}

type TopicPublishEvent struct {
	Payload string `json:"payload"`
}

type TopicSubscriberResultEvent struct {
	Success bool `json:"success"`
}

type EventRecord struct {
	EventId  string `json:"eventId,omitempty"`
	TopicKey string `json:"topicKey,omitempty"`
}

type EventResponseRecord struct {
	Success bool
}

type EventRequestRecord struct{}

type EventHistoryItem struct {
	Event     *EventRecord `json:"event,omitempty"`
	Request   *EventRequestRecord
	Payload   string `json:"payload,omitempty"`
	Responses *EventResponseRecord
	// Responses []bool `json:"responses,omitempty"`
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
	projectDir string
}

func NewHistory(projectDir string) *History {
	h := &History{
		projectDir: projectDir,
	}

	// Start the goroutine to handle write operation
	_ = eventbus.Bus().Subscribe(AddRecordTopic, h.writeHistoryRecord)

	return h
}

func NewHistoryError(recordType RecordType, historyFile string) error {
	return fmt.Errorf("could not write %s history to the JSON file '%s' due to a formatting issue. Please check the file's formatting and ensure it follows the correct JSON structure, or reset the history by deleting the file", recordType, historyFile)
}

func (h *History) writeHistoryRecord(historyRecord *HistoryEvent[any]) error {
	historyFile, err := utils.NitricHistoryFile(h.projectDir, string(historyRecord.RecordType))
	if err != nil {
		return err
	}

	existingRecords, err := ReadHistoryRecords[any](h, historyRecord.RecordType)
	if err != nil {
		return NewHistoryError(historyRecord.RecordType, historyFile)
	}

	existingRecords = append(existingRecords, historyRecord)

	data, err := json.Marshal(existingRecords)
	if err != nil {
		return NewHistoryError(historyRecord.RecordType, historyFile)
	}

	err = os.WriteFile(historyFile, data, fs.ModePerm)
	if err != nil {
		return NewHistoryError(historyRecord.RecordType, historyFile)
	}

	return nil
}

func (h *History) DeleteHistoryRecord(recordType RecordType) error {
	historyFile, err := utils.NitricHistoryFile(h.projectDir, string(recordType))
	if err != nil {
		return err
	}

	return os.Remove(historyFile)
}

func (h *History) ReadAllHistoryRecords() (*HistoryEvents, error) {
	schedules, err := ReadHistoryRecords[any](h, SCHEDULE)
	if err != nil {
		return nil, fmt.Errorf("error occurred reading schedule history: %w", err)
	}

	topics, err := ReadHistoryRecords[TopicEvent](h, TOPIC)
	if err != nil {
		return nil, fmt.Errorf("error occurred reading topic history: %w", err)
	}

	apis, err := ReadHistoryRecords[ApiHistoryItem](h, API)
	if err != nil {
		return nil, fmt.Errorf("error occurred reading api history: %w", err)
	}

	return &HistoryEvents{
		ScheduleHistory: schedules,
		TopicHistory:    topics,
		ApiHistory:      apis,
	}, nil
}

func ReadHistoryRecords[T any](h *History, recordType RecordType) ([]*HistoryEvent[T], error) {
	historyFile, err := utils.NitricHistoryFile(h.projectDir, string(recordType))
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(historyFile)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return []*HistoryEvent[T]{}, nil
	}

	var history []*HistoryEvent[T]

	err = json.Unmarshal(data, &history)
	if err != nil {
		// Check if the error is a JSON syntax error
		if _, ok := err.(*json.SyntaxError); ok {
			return nil, fmt.Errorf("JSON syntax issue detected.\nTo fix, delete the file '%s'", historyFile)
		}

		return nil, err
	}

	return history, nil
}
