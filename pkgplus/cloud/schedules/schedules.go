package schedules

import (
	"fmt"
	"log"
	"maps"
	"strconv"
	"strings"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/nitrictech/cli/pkgplus/streams"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/schedules"
	"github.com/robfig/cron/v3"
)

type State = map[string]*schedulespb.RegistrationRequest

type LocalSchedulesService struct {
	*schedules.ScheduleWorkerManager
	cron *cron.Cron

	schedulesLock sync.RWMutex

	schedules State
	bus       EventBus.Bus
}

const localSchedulesTopic = "local_schedules"

func (l *LocalSchedulesService) publishState() {
	l.bus.Publish(localSchedulesTopic, maps.Clone(l.schedules))
}

func (l *LocalSchedulesService) SubscribeToState(fn func(State)) {
	l.bus.Subscribe(localSchedulesTopic, fn)
}

var _ schedulespb.SchedulesServer = (*LocalSchedulesService)(nil)

func (l *LocalSchedulesService) GetSchedules() State {
	l.schedulesLock.RLock()
	defer l.schedulesLock.RUnlock()

	return l.schedules
}

func (l *LocalSchedulesService) registerSchedule(registrationRequest *schedulespb.RegistrationRequest) {
	l.schedulesLock.Lock()
	defer l.schedulesLock.Unlock()

	l.schedules[registrationRequest.ScheduleName] = registrationRequest
}

func (l *LocalSchedulesService) createCronSchedule(scheduleName, expression string) (cron.EntryID, error) {
	return l.cron.AddFunc(expression, func() {
		l.HandleRequest(&schedulespb.ServerMessage{
			Id: "",
			Content: &schedulespb.ServerMessage_IntervalRequest{
				IntervalRequest: &schedulespb.IntervalRequest{
					ScheduleName: scheduleName,
				},
			},
		})
	})
}

func (l *LocalSchedulesService) Schedule(stream schedulespb.Schedules_ScheduleServer) error {
	peekableStream := streams.NewPeekableStreamServer[*schedulespb.ServerMessage, *schedulespb.ClientMessage](stream)

	firstRequest, err := peekableStream.Peek()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		return fmt.Errorf("first request must be a registration request")
	}

	l.registerSchedule(firstRequest.GetRegistrationRequest())

	scheduleName := firstRequest.GetRegistrationRequest().ScheduleName
	cronExpression := ""
	switch t := firstRequest.GetRegistrationRequest().Cadence.(type) {
	case *schedulespb.RegistrationRequest_Cron:
		cronExpression = t.Cron.Expression
	case *schedulespb.RegistrationRequest_Every:
		parts := strings.Split(strings.TrimSpace(t.Every.Rate), " ")
		if len(parts) != 2 {
			return fmt.Errorf("invalid schedule rate: %s", t.Every.Rate)
		}

		initialRate, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid schedule rate, must start with an integer")
		}

		// Dapr cron bindings only support hours, minutes and seconds. Convert days to hours
		if strings.HasPrefix(parts[1], "day") {
			parts[0] = fmt.Sprintf("%d", initialRate*24)
			parts[1] = "hours"
		}

		cronExpression = fmt.Sprintf("@every %s%c", parts[0], parts[1][0])
	default:
		return fmt.Errorf("unknown schedule type, must be one of: cron, every")
	}

	fmt.Println(cronExpression)

	cronEntryId, err := l.createCronSchedule(scheduleName, cronExpression)
	if err != nil {
		return err
	}

	defer l.cron.Remove(cronEntryId)

	if err != nil {
		log.Fatal(err)
	}

	// Start the cron scheduler
	l.cron.Start()

	return l.ScheduleWorkerManager.Schedule(peekableStream)
}

func NewLocalSchedulesService() *LocalSchedulesService {
	return &LocalSchedulesService{
		ScheduleWorkerManager: schedules.New(),
		cron:                  cron.New(),
		bus:                   EventBus.New(),
		schedules:             make(map[string]*schedulespb.RegistrationRequest),
	}
}
