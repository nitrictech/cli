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
	"fmt"
	"strings"

	"github.com/pterm/pterm"

	"github.com/nitrictech/cli/pkg/preview"
	"github.com/nitrictech/cli/pkg/project"
	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
	"github.com/nitrictech/nitric/core/pkg/worker"
	"github.com/nitrictech/nitric/core/pkg/worker/pool"
)

type BucketNotification struct {
	Bucket                   string
	NotificationType         v1.BucketNotificationType
	NotificationPrefixFilter string
}

type RunStackState struct {
	project             *project.Project
	apis                map[string]string
	subs                map[string]string
	schedules           map[string]string
	bucketNotifications []*BucketNotification
	httpWorkers         map[int]string
}

func (r *RunStackState) Update(workerPool pool.WorkerPool, ls LocalServices) {
	// reset state maps
	r.apis = make(map[string]string)
	r.subs = make(map[string]string)
	r.schedules = make(map[string]string)
	r.bucketNotifications = []*BucketNotification{}
	r.httpWorkers = make(map[int]string)

	for name, address := range ls.Apis() {
		r.apis[name] = address
	}

	for port, address := range ls.HttpWorkers() {
		r.httpWorkers[port] = address
	}

	// TODO: We can probably move this directly into local service state
	for _, wrkr := range workerPool.GetWorkers(&pool.GetWorkerOptions{}) {
		switch w := wrkr.(type) {
		case *worker.SubscriptionWorker:
			r.subs[w.Topic()] = fmt.Sprintf("http://%s/topic/%s", ls.TriggerAddress(), w.Topic())
		case *worker.ScheduleWorker:
			topicKey := strings.ToLower(strings.ReplaceAll(w.Key(), " ", "-"))
			r.subs[w.Key()] = fmt.Sprintf("http://%s/topic/%s", ls.TriggerAddress(), topicKey)
		case *worker.BucketNotificationWorker:
			r.bucketNotifications = append(r.bucketNotifications, &BucketNotification{
				Bucket:                   w.Bucket(),
				NotificationType:         w.NotificationType(),
				NotificationPrefixFilter: w.NotificationPrefixFilter(),
			})
		}
	}
}

func (r *RunStackState) Warnings() []string {
	warnings := []string{}

	if !r.project.IsPreviewFeatureEnabled(preview.Feature_Http) && len(r.httpWorkers) > 0 {
		warnings = append(warnings, "You are using a preview feature 'http' before deploying you will need to enable this in your project file.")
	}

	return warnings
}

func (r *RunStackState) Tables(port int, dashPort int) (string, error) {
	tables := []string{}

	table, rows, err := r.ApiTable(9001)
	if err != nil {
		return "", err
	}

	if rows > 0 {
		tables = append(tables, table)
	}

	table, rows, err = r.TopicTable()
	if err != nil {
		return "", err
	}

	if rows > 0 {
		tables = append(tables, table)
	}

	table, rows, err = r.SchedulesTable()
	if err != nil {
		return "", err
	}

	if rows > 0 {
		tables = append(tables, table)
	}

	table, rows, err = r.BucketNotificationsTable()
	if err != nil {
		return "", err
	}

	if rows > 0 {
		tables = append(tables, table)
	}

	table, rows, err = r.HttpTable()
	if err != nil {
		return "", err
	}

	if rows > 0 {
		tables = append(tables, table)
	}

	table, err = r.DashboardTable(dashPort)
	if err != nil {
		return "", err
	}

	tables = append(tables, table)

	return strings.Join(tables, "\n\n"), nil
}

func (r *RunStackState) ApiTable(port int) (string, int, error) {
	tableData := pterm.TableData{{"Api", "Endpoint"}}

	for name, address := range r.apis {
		tableData = append(tableData, []string{
			name, fmt.Sprintf("http://%s", address),
		})
	}

	str, err := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()
	if err != nil {
		return "", 0, err
	}

	return str, len(r.apis), nil
}

func (r *RunStackState) TopicTable() (string, int, error) {
	tableData := pterm.TableData{{"Topic", "Endpoint"}}

	for k, address := range r.subs {
		tableData = append(tableData, []string{
			k, address,
		})
	}

	str, err := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()
	if err != nil {
		return "", 0, err
	}

	return str, len(r.subs), nil
}

func (r *RunStackState) SchedulesTable() (string, int, error) {
	tableData := pterm.TableData{{"Schedule", "Endpoint"}}

	for k, address := range r.schedules {
		tableData = append(tableData, []string{
			k, address,
		})
	}

	str, err := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()
	if err != nil {
		return "", 0, err
	}

	return str, len(r.schedules), nil
}

func (r *RunStackState) HttpTable() (string, int, error) {
	tableData := pterm.TableData{{"Proxy", "Endpoint"}}

	for port, address := range r.httpWorkers {
		tableData = append(tableData, []string{
			fmt.Sprintf("%d", port), fmt.Sprintf("http://%s", address),
		})
	}

	str, err := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()
	if err != nil {
		return "", 0, err
	}

	return str, len(r.httpWorkers), nil
}

func (r *RunStackState) BucketNotificationsTable() (string, int, error) {
	tableData := pterm.TableData{{"Bucket", "Notification Type", "Notification Prefix Filter"}}

	for _, notification := range r.bucketNotifications {
		tableData = append(tableData, []string{
			notification.Bucket, notification.NotificationType.String(), notification.NotificationPrefixFilter,
		})
	}

	str, err := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()
	if err != nil {
		return "", 0, err
	}

	return str, len(r.bucketNotifications), nil
}

func (r *RunStackState) DashboardTable(port int) (string, error) {
	tableData := pterm.TableData{{pterm.LightCyan("Dev Dashboard"), fmt.Sprintf("http://localhost:%v", port)}}

	str, err := pterm.DefaultTable.WithData(tableData).Srender()
	if err != nil {
		return "", err
	}

	return str, nil
}

func NewStackState(proj *project.Project) *RunStackState {
	return &RunStackState{
		project:             proj,
		apis:                map[string]string{},
		subs:                map[string]string{},
		schedules:           map[string]string{},
		bucketNotifications: []*BucketNotification{},
		httpWorkers:         map[int]string{},
	}
}
