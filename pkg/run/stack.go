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
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"

	"github.com/nitrictech/cli/pkg/preview"
	"github.com/nitrictech/cli/pkg/project"
	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
	"github.com/nitrictech/nitric/core/pkg/worker"
	"github.com/nitrictech/nitric/core/pkg/worker/pool"
	"github.com/nitrictech/pearls/pkg/tui"
)

const (
	urlWidth     = 35
	defaultWidth = 15
)

type BucketNotification struct {
	Bucket                   string
	NotificationType         v1.BucketNotificationType
	NotificationPrefixFilter string
}

type RunStackState struct {
	project             *project.Project
	apis                map[string]string
	sockets             map[string]string
	subs                map[string]string
	schedules           map[string]string
	bucketNotifications []*BucketNotification
	httpWorkers         map[int]string
	dashboardPort       int
}

func (r *RunStackState) Update(workerPool pool.WorkerPool, ls LocalServices) {
	// reset state maps
	r.apis = make(map[string]string)
	r.subs = make(map[string]string)
	r.sockets = make(map[string]string)
	r.schedules = make(map[string]string)
	r.bucketNotifications = []*BucketNotification{}
	r.httpWorkers = make(map[int]string)
	r.dashboardPort = ls.GetDashboard().GetPort()

	for name, address := range ls.Apis() {
		r.apis[name] = address
	}

	for port, address := range ls.HttpWorkers() {
		r.httpWorkers[port] = address
	}

	for name, address := range ls.Websockets() {
		r.sockets[name] = address
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

	if !r.project.IsPreviewFeatureEnabled(preview.Feature_Websockets) && len(r.sockets) > 0 {
		warnings = append(warnings, "You are using a preview feature 'websockets' before deploying you will need to enable this in your project file.")
	}

	return warnings
}

func createTable(columns []table.Column, rows []table.Row) table.Model {
	headerStyle := lipgloss.NewStyle().Bold(true)
	headers := []table.Column{}

	for _, column := range columns {
		headers = append(headers, table.Column{Title: headerStyle.Render(column.Title), Width: column.Width})
	}

	t := table.New(
		table.WithColumns(headers),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)+1),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(tui.Colors.White).
		BorderBottom(true).
		Bold(true)
	s.Selected = lipgloss.NewStyle()
	t.SetStyles(s)

	return t
}

func (r *RunStackState) Tables() []table.Model {
	port := 9001
	tables := []table.Model{}

	table, rows := r.ApiTable(port)
	if rows > 0 {
		tables = append(tables, table)
	}

	table, rows = r.WebsocketsTable(port)
	if rows > 0 {
		tables = append(tables, table)
	}

	table, rows = r.TopicTable()
	if rows > 0 {
		tables = append(tables, table)
	}

	table, rows = r.SchedulesTable()
	if rows > 0 {
		tables = append(tables, table)
	}

	table, rows = r.BucketNotificationsTable()
	if rows > 0 {
		tables = append(tables, table)
	}

	table, rows = r.HttpTable()
	if rows > 0 {
		tables = append(tables, table)
	}

	// Only add the dashboard table if theres more resources
	table = r.DashboardTable(r.dashboardPort)
	if len(tables) > 0 {
		tables = append(tables, table)
	}

	return tables
}

func (r *RunStackState) ApiTable(port int) (table.Model, int) {
	columns := []table.Column{
		{Title: "Api", Width: defaultWidth},
		{Title: "Endpoint", Width: urlWidth},
	}
	rows := make([]table.Row, 0)

	for name, address := range r.apis {
		rows = append(rows, []string{
			name, fmt.Sprintf("http://%s", address),
		})
	}

	return createTable(columns, rows), len(rows)
}

func (r *RunStackState) WebsocketsTable(port int) (table.Model, int) {
	columns := []table.Column{
		{Title: "Websocket", Width: defaultWidth},
		{Title: "Endpoint", Width: urlWidth},
	}
	rows := make([]table.Row, 0)

	for name, address := range r.sockets {
		rows = append(rows, []string{
			name, fmt.Sprintf("ws://%s", address),
		})
	}

	return createTable(columns, rows), len(rows)
}

func (r *RunStackState) TopicTable() (table.Model, int) {
	columns := []table.Column{
		{Title: "Topic", Width: defaultWidth},
		{Title: "Endpoint", Width: urlWidth},
	}
	rows := make([]table.Row, 0)

	topicKeys := lo.Keys(r.subs)
	sort.Strings(topicKeys)

	for _, k := range topicKeys {
		rows = append(rows, []string{
			k, r.subs[k],
		})
	}

	return createTable(columns, rows), len(r.subs)
}

func (r *RunStackState) SchedulesTable() (table.Model, int) {
	columns := []table.Column{
		{Title: "Schedule", Width: defaultWidth},
		{Title: "Endpoint", Width: urlWidth},
	}
	rows := make([]table.Row, 0)

	for k, address := range r.schedules {
		rows = append(rows, []string{
			k, address,
		})
	}

	return createTable(columns, rows), len(r.schedules)
}

func (r *RunStackState) HttpTable() (table.Model, int) {
	columns := []table.Column{
		{Title: "Proxy", Width: defaultWidth},
		{Title: "Endpoint", Width: urlWidth},
	}
	rows := make([]table.Row, 0)

	for port, address := range r.httpWorkers {
		rows = append(rows, []string{
			fmt.Sprintf("%d", port), fmt.Sprintf("http://%s", address),
		})
	}

	return createTable(columns, rows), len(r.httpWorkers)
}

func (r *RunStackState) BucketNotificationsTable() (table.Model, int) {
	columns := []table.Column{
		{Title: "Bucket", Width: defaultWidth},
		{Title: "Type", Width: defaultWidth},
		{Title: "Filter", Width: defaultWidth},
	}
	rows := make([]table.Row, 0)

	for _, notification := range r.bucketNotifications {
		rows = append(rows, []string{
			notification.Bucket, notification.NotificationType.String(), notification.NotificationPrefixFilter,
		})
	}

	return createTable(columns, rows), len(r.bucketNotifications)
}

func (r *RunStackState) DashboardTable(port int) table.Model {
	columns := []table.Column{{Title: "Dev Dashboard", Width: urlWidth}}
	rows := make([]table.Row, 0)

	rows = append(rows, []string{fmt.Sprintf("http://localhost:%v", port)})

	return createTable(columns, rows)
}

func NewStackState(proj *project.Project) *RunStackState {
	return &RunStackState{
		project:             proj,
		apis:                map[string]string{},
		sockets:             map[string]string{},
		subs:                map[string]string{},
		schedules:           map[string]string{},
		bucketNotifications: []*BucketNotification{},
		httpWorkers:         map[int]string{},
	}
}
