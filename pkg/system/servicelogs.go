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

package system

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nitrictech/cli/pkg/paths"
	"github.com/sirupsen/logrus"
)

type LogEntry struct {
	Timestamp   string       `json:"time"`
	Level       logrus.Level `json:"level"`
	Message     string       `json:"msg"`
	ServiceName string       `json:"serviceName"`
}

// ServiceLogger struct to encapsulate the logger and file path
type ServiceLogger struct {
	Logger      *logrus.Logger
	LogFilePath string
}

// NewServiceLogger creates a new instance of ServiceLogger with the specified log file path
func NewServiceLogger(stackFilePath string) *ServiceLogger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &ServiceLogger{
		Logger:      logger,
		LogFilePath: paths.NitricServiceLogFile(stackFilePath),
	}
}

// WriteLog writes a log entry with the specified level and message
func (s *ServiceLogger) WriteLog(level logrus.Level, message, serviceName string) error {
	// Open the log file when writing a log entry
	file, err := os.OpenFile(s.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not open log file: %v", err)
	}
	defer file.Close() // Ensure the file is closed after writing

	// Set the output of the logger to the file
	s.Logger.SetOutput(file)

	s.Logger.WithFields(logrus.Fields{
		"serviceName": serviceName,
	}).Log(level, message)

	return nil
}

// ReadLogs reads the log file from the service's log file path and returns a slice of LogEntry objects
func ReadLogs(stackFilePath string) ([]LogEntry, error) {
	// Open the log file for reading
	file, err := os.Open(paths.NitricServiceLogFile(stackFilePath))
	if err != nil {
		return nil, fmt.Errorf("could not open log file: %v", err)
	}
	defer file.Close() // Ensure the file is closed when the function finishes

	var logs []LogEntry

	// Read the file line by line
	decoder := json.NewDecoder(file)
	for {
		var log LogEntry
		if err := decoder.Decode(&log); err != nil {
			if err.Error() == "EOF" {
				break // End of file reached
			}
			return nil, fmt.Errorf("error decoding log entry: %v", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// PurgeLogs deletes the log file from the service's log file path
func PurgeLogs(stackFilePath string) error {
	// Remove the log file
	err := os.Remove(paths.NitricServiceLogFile(stackFilePath))
	if err != nil {
		return fmt.Errorf("could not remove log file: %v", err)
	}

	return nil
}
