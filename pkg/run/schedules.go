package run

import (
	"fmt"
	"log"
	"sync"

	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/schedules"
	"github.com/robfig/cron/v3"
)

type LocalSchedules struct {
	*schedules.ScheduleWorkerManager
	cron *cron.Cron

	schedulesLock sync.RWMutex

	schedules map[string]*schedulespb.RegistrationRequest
}

var _ schedulespb.SchedulesServer = (*LocalSchedules)(nil)

func (l *LocalSchedules) GetSchedules() map[string]*schedulespb.RegistrationRequest {
	l.schedulesLock.RLock()
	defer l.schedulesLock.RUnlock()

	return l.schedules
}

func (l *LocalSchedules) registerSchedule(registrationRequest *schedulespb.RegistrationRequest) {
	l.schedulesLock.Lock()
	defer l.schedulesLock.Unlock()

	l.schedules[registrationRequest.ScheduleName] = registrationRequest
}

func (l *LocalSchedules) Schedule(stream schedulespb.Schedules_ScheduleServer) error {
	peekableStream := NewPeekableStreamServer[*schedulespb.ServerMessage, *schedulespb.ClientMessage](stream)

	firstRequest, err := peekableStream.Peek()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		return fmt.Errorf("first request must be a registration request")
	}

	l.registerSchedule(firstRequest.GetRegistrationRequest())

	// TODO: add support for rates - convert to cron
	if firstRequest.GetRegistrationRequest().GetCron() != nil {
		// Schedule your task and provide the callback function
		cronEntryId, err := l.cron.AddFunc(firstRequest.GetRegistrationRequest().GetCron().Expression, func() {
			l.HandleRequest(&schedulespb.ServerMessage{
				Id: "",
				Content: &schedulespb.ServerMessage_IntervalRequest{
					IntervalRequest: &schedulespb.IntervalRequest{
						ScheduleName: firstRequest.GetRegistrationRequest().ScheduleName,
					},
				},
			})
		})
		if err != nil {
			return err
		}

		defer l.cron.Remove(cronEntryId)
	}

	if err != nil {
		log.Fatal(err)
	}

	// Start the cron scheduler
	l.cron.Start()

	return l.ScheduleWorkerManager.Schedule(peekableStream)
}

func NewLocalSchedules() *LocalSchedules {
	return &LocalSchedules{
		ScheduleWorkerManager: schedules.New(),
		cron:                  cron.New(),
	}
}
