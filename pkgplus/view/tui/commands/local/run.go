package local

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nitrictech/cli/pkgplus/cloud"
	"github.com/nitrictech/cli/pkgplus/cloud/apis"
	"github.com/nitrictech/cli/pkgplus/cloud/http"
	"github.com/nitrictech/cli/pkgplus/cloud/resources"
	"github.com/nitrictech/cli/pkgplus/cloud/schedules"
	"github.com/nitrictech/cli/pkgplus/cloud/topics"
	"github.com/nitrictech/cli/pkgplus/cloud/websockets"
	tui "github.com/nitrictech/cli/pkgplus/view/tui/components"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
	viewr "github.com/nitrictech/cli/pkgplus/view/tui/components/view"
	"github.com/nitrictech/cli/pkgplus/view/tui/reactive"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
)

type ApiSummary struct {
	Name               string
	Url                string
	RequestingServices []string
}

type WebsocketSummary struct {
	name string
	url  string
}

type HttpProxySummary struct {
	name string
	url  string
}

type TopicSummary struct {
	name            string
	url             string
	subscriberCount int
}

type ScheduleSummary struct {
	name string
	rate string
	url  string
}

type TuiModel struct {
	localCloud  *cloud.LocalCloud
	apis        []ApiSummary
	websockets  []WebsocketSummary
	httpProxies []HttpProxySummary
	topics      []TopicSummary
	schedules   []ScheduleSummary

	resources *resources.LocalResourcesState

	reactiveSub *reactive.Subscription

	dashboardUrl string
}

const addDashboardUrlTopic = "add_dashboard_url"

var _ tea.Model = &TuiModel{}

func (t *TuiModel) Init() tea.Cmd {
	t.reactiveSub = reactive.NewSubscriber()
	reactive.ListenFor(t.reactiveSub, t.localCloud.Apis.SubscribeToState)
	reactive.ListenFor(t.reactiveSub, t.localCloud.Websockets.SubscribeToState)
	reactive.ListenFor(t.reactiveSub, t.localCloud.Http.SubscribeToState)

	reactive.ListenFor(t.reactiveSub, t.localCloud.Resources.SubscribeToState)

	reactive.ListenFor(t.reactiveSub, t.localCloud.Schedules.SubscribeToState)
	reactive.ListenFor(t.reactiveSub, t.localCloud.Topics.SubscribeToState)

	return t.reactiveSub.AwaitNextMsg()
}

func (t *TuiModel) ReactiveUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch state := msg.(type) {
	case resources.LocalResourcesState:
		t.resources = &state
	case apis.State:
		// update the api state by getting the latest API addresses
		newApiSummary := []ApiSummary{}

		for apiName, serviceReg := range state {
			reqSrvs := []string{}
			for srv := range serviceReg {
				reqSrvs = append(reqSrvs, srv)
			}

			newApiSummary = append(newApiSummary, ApiSummary{
				Name:               apiName,
				Url:                t.localCloud.Gateway.GetApiAddresses()[apiName],
				RequestingServices: reqSrvs,
			})
		}

		t.apis = newApiSummary
	case websockets.State:
		// update the api state by getting the latest API addresses
		newWebsocketsSummary := []WebsocketSummary{}

		for api, host := range t.localCloud.Gateway.GetWebsocketAddresses() {
			newWebsocketsSummary = append(newWebsocketsSummary, WebsocketSummary{
				name: api,
				url:  host,
			})
		}

		t.websockets = newWebsocketsSummary
	case http.State:
		// update the api state by getting the latest API addresses
		newHttpProxiesSummary := []HttpProxySummary{}

		for api, host := range t.localCloud.Gateway.GetHttpWorkerAddresses() {
			newHttpProxiesSummary = append(newHttpProxiesSummary, HttpProxySummary{
				name: api,
				url:  host,
			})
		}

		t.httpProxies = newHttpProxiesSummary
	case topics.State:
		// update the api state by getting the latest API addresses
		newTopicsSummary := []TopicSummary{}

		for topic, subscribedService := range state {
			// Each service can subscribe more than once.
			subCount := 0
			for _, numSubscribers := range subscribedService {
				subCount += numSubscribers
			}

			newTopicsSummary = append(newTopicsSummary, TopicSummary{
				name:            topic,
				url:             t.localCloud.Gateway.GetTopicTriggerUrl(topic),
				subscriberCount: subCount,
			})
		}

		t.topics = newTopicsSummary
	case schedules.State:
		// update the api state by getting the latest API addresses
		newSchedulesSummary := []ScheduleSummary{}

		for schedule, scheduledService := range state {
			var rate string = ""

			switch t := scheduledService.Schedule.Cadence.(type) {
			case *schedulespb.RegistrationRequest_Cron:
				rate = t.Cron.Expression
			case *schedulespb.RegistrationRequest_Every:
				rate = t.Every.Rate
			default:
				rate = "unknown"
			}

			newSchedulesSummary = append(newSchedulesSummary, ScheduleSummary{
				name: schedule,
				url:  t.localCloud.Gateway.GetScheduleManualTriggerUrl(schedule),
				rate: rate,
			})
		}

		t.schedules = newSchedulesSummary
	}

	return t, t.reactiveSub.AwaitNextMsg()
}

func (t *TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch typ := msg.(type) {
	case tea.KeyMsg:
		keyMsg := msg.(tea.KeyMsg)

		switch keyMsg.String() {
		case "ctrl+c", "q":
			return t, tea.Quit
		}

	// Using a wrapper here
	case reactive.Message:
		return t.ReactiveUpdate(typ.Msg)
	default:
		break
	}

	return t, tea.Batch(cmds...)
}

var (
	textStyle = lipgloss.NewStyle().Foreground(tui.Colors.White).Align(lipgloss.Left)
	// TODO: Extract into common title styles
	titleStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(tui.Colors.White).
			Background(tui.Colors.Blue).
			MarginRight(2)
	tagStyle = lipgloss.NewStyle().Width(10).Background(tui.Colors.Purple).Foreground(tui.Colors.White)
)

func (t *TuiModel) View() string {
	output := viewr.New(view.WithStyle(textStyle))

	output.Addln("Nitric").WithStyle(titleStyle)
	output.Break()

	apisRegistered := len(t.apis) > 0
	websocketsRegistered := len(t.websockets) > 0
	httpProxiesRegistered := len(t.httpProxies) > 0
	topicsRegistered := len(t.topics) > 0
	schedulesRegistered := len(t.schedules) > 0

	noWorkersRegistered := !apisRegistered && !websocketsRegistered && !httpProxiesRegistered && !topicsRegistered && !schedulesRegistered

	if t.dashboardUrl != "" && !noWorkersRegistered {
		output.Add("Dashboard: ").WithStyle(tagStyle)
		output.Addln(t.dashboardUrl).WithStyle(lipgloss.NewStyle().Bold(true))
		output.Break()
	}

	// Show APIs
	if apisRegistered {
		output.Addln("APIs:").WithStyle(tagStyle)
		output.Break()

		for _, api := range t.apis {
			output.Add(api.Name).WithStyle(lipgloss.NewStyle().Bold(true))
			output.Addln(" => %s", api.Url)
			output.Addln(" for %s", strings.Join(api.RequestingServices, ", "))
			output.Break()
		}
	}

	// Show HTTP Servers
	if httpProxiesRegistered {
		output.Addln("HTTP Servers:").WithStyle(tagStyle)
		output.Break()

		for _, httpProxy := range t.httpProxies {
			output.Add(httpProxy.name).WithStyle(lipgloss.NewStyle().Bold(true))
			output.Addln(" => %s", httpProxy.url)
			output.Break()
		}
	}

	// Show APIs
	if websocketsRegistered {
		output.Addln("Websockets:").WithStyle(tagStyle)
		output.Break()

		for _, websocket := range t.websockets {
			output.Add(websocket.name).WithStyle(lipgloss.NewStyle().Bold(true))
			output.Addln(" => %s", websocket.url)
			output.Break()
		}
	}

	if topicsRegistered {
		output.Addln("Topics:").WithStyle(tagStyle)
		output.Break()

		for _, topic := range t.topics {
			output.Add(topic.name).WithStyle(lipgloss.NewStyle().Bold(true))
			output.Addln(" => %s (%d subscribers)", topic.url, topic.subscriberCount)
			output.Break()
		}
	}

	if schedulesRegistered {
		output.Addln("Schedules:").WithStyle(tagStyle)
		output.Break()

		for _, schedule := range t.schedules {
			output.Add(schedule.name).WithStyle(lipgloss.NewStyle().Bold(true))
			output.Addln(" => %s (%s)", schedule.url, schedule.rate)
			output.Break()
		}
	}

	// Show waiting message if no workers are connected
	if noWorkersRegistered {
		output.Addln("waiting for connections, start your application to connect it with the local nitric server.").WithStyle(lipgloss.NewStyle().Bold(true))
		output.Break()
	}

	// // Render resources
	// if t.resources != nil {
	// 	output.Addln("Resources:").WithStyle(tagStyle)
	// 	output.Break()

	// 	for name, bucket := range t.resources.Buckets.GetAll() {
	// 		output.Addln("Bucket::%s", name)
	// 		output.Addln(" for %s", strings.Join(bucket.RequestingServices, ", "))
	// 	}

	// 	for name, policy := range t.resources.Policies.GetAll() {
	// 		output.Addln("Policy::%s", name)
	// 		output.Addln(" - %+v", policy.Resource)
	// 	}
	// }

	// Show relevant links
	return output.Render()
}

func NewTuiModel(localCloud *cloud.LocalCloud, dashboardUrl string) *TuiModel {
	return &TuiModel{
		localCloud:   localCloud,
		dashboardUrl: dashboardUrl,
	}
}
