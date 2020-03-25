package actors

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	activeReminders reminderQueue
	arMutex         = sync.Mutex{}
)

func init() {
	activeReminders = make(reminderQueue, 0)
	heap.Init(&activeReminders)
}

// Reminder is a reminder
type Reminder struct {
	ActorType string        `json:"actorType"`
	ActorID   string        `json:"actorId"`
	ID        string        `json:"id"`
	Deadline  time.Time     `json:"deadline"`
	Period    time.Duration `json:"period,omitempty"` // 0 for one-shot reminders
	Data      interface{}   `json:"data,omitempty"`
}

// CancelReminderPayload is the JSON request body for cancelling a reminder
type CancelReminderPayload struct {
	ID string `json:"id"`
}

// GetReminderPayload is the JSON request body for getting a reminder
type GetReminderPayload struct {
	ID string `json:"id"`
}

// ScheduleReminderPayload is the JSON request body for scheduling a new reminder
type ScheduleReminderPayload struct {
	ID       string      `json:"id"`
	Deadline time.Time   `json:"deadline"`
	Period   string      `json:"period,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

// CancelReminder attempts to cancel the argument reminder
func CancelReminder(actorType string, actorID string, payload CancelReminderPayload) (bool, error) {
	return true, nil
}

// GetReminders returns all reminders that match the provided filter
func GetReminders(actorType string, actorID string, payload GetReminderPayload) ([]Reminder, error) {
	return nil, nil
}

// ScheduleReminder schedules a new reminder
func ScheduleReminder(actorType string, actorID string, payload ScheduleReminderPayload) (validRequest bool, err error) {
	r := Reminder{
		ActorType: actorType,
		ActorID:   actorID,
		ID:        payload.ID,
		Deadline:  payload.Deadline,
	}
	if payload.Period != "" {
		period, err := time.ParseDuration(payload.Period)
		if err != nil {
			return false, err
		}
		r.Period = period
	}

	logger.Info("ScheduleReminder: %v", r)
	arMutex.Lock()
	activeReminders.addReminder(r)
	arMutex.Unlock()

	return true, nil
}

// ProcessReminders causes all reminders with a deadline before time to be scheduled for execution.
func ProcessReminders(ctx context.Context, fireTime time.Time) {
	logger.Debug("ProcessReminders invoked at %v", fireTime)
	arMutex.Lock()

	for {
		r, valid := activeReminders.nextReminderBefore(fireTime)
		if !valid {
			break
		}
		logger.Info("ProcessReminders: at %v firing %v (deadline %v)", fireTime, r.ID, r.Deadline)
		if r.Period > 0 {
			r.Deadline = fireTime.Add(r.Period)
			activeReminders.addReminder(r)
		}
	}

	arMutex.Unlock()
	logger.Debug("Completed reminder processing for time %v", fireTime)
}
