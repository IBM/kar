package actors

import (
	"container/heap"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	activeReminders        reminderQueue
	arMutex                sync.Mutex
	jitterWarningThreshold time.Duration
)

func init() {
	activeReminders = make(reminderQueue, 0)
	heap.Init(&activeReminders)
	jitterWarningThreshold = time.Duration(config.ActorReminderAcceptableJitterFactor * int64(config.ActorReminderInterval))
}

// Reminder is a reminder
type Reminder struct {
	Actor       Actor
	ID          string        `json:"id"`
	Path        string        `json:"path"`
	Deadline    time.Time     `json:"deadline"`
	Period      time.Duration `json:"period,omitempty"` // 0 for one-shot reminders
	EncodedData string        `json:"encodedData,omitempty"`
}

// CancelReminderPayload is the JSON request body for cancelling a reminder
type CancelReminderPayload struct {
	ID string `json:"id,omitempty"`
}

// GetReminderPayload is the JSON request body for getting a reminder
type GetReminderPayload struct {
	ID string `json:"id,omitempty"`
}

// ScheduleReminderPayload is the JSON request body for scheduling a new reminder
type ScheduleReminderPayload struct {
	ID       string      `json:"id"`
	Path     string      `json:"path"`
	Deadline time.Time   `json:"deadline"`
	Period   string      `json:"period,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

// CancelReminder attempts to cancel the argument reminder
func CancelReminder(actor Actor, payload CancelReminderPayload) (bool, error) {
	arMutex.Lock()
	found := activeReminders.cancelReminder(actor, payload.ID)
	arMutex.Unlock()
	if found {
		logger.Debug("Cancelled reminders matching (%v, %v)", actor, payload.ID)
	}
	return found, nil
}

// GetReminders returns all reminders that match the provided filter
func GetReminders(actor Actor, payload GetReminderPayload) ([]Reminder, error) {
	arMutex.Lock()
	found := activeReminders.findMatchingReminders(actor, payload.ID)
	arMutex.Unlock()
	return found, nil
}

// ScheduleReminder schedules a new reminder
func ScheduleReminder(actor Actor, payload ScheduleReminderPayload) (validRequest bool, err error) {
	r := Reminder{
		Actor:    actor,
		Path:     payload.Path,
		ID:       payload.ID,
		Deadline: payload.Deadline,
	}
	if payload.Period != "" {
		period, err := time.ParseDuration(payload.Period)
		if err != nil {
			return false, err
		}
		r.Period = period
	}
	if payload.Data != nil {
		buf, err := json.Marshal(payload.Data)
		if err != nil {
			return false, err
		}
		r.EncodedData = string(buf)
	}

	logger.Info("ScheduleReminder: %v", r)
	arMutex.Lock()
	activeReminders.addReminder(r)
	arMutex.Unlock()

	return true, nil
}

// ProcessReminders causes all reminders with a deadline before fireTime to be scheduled for execution.
func ProcessReminders(ctx context.Context, fireTime time.Time, tell func(context context.Context, actor Actor, path, payload, contentType string) error) {
	logger.Debug("ProcessReminders invoked at %v", fireTime)
	arMutex.Lock()

	for {
		r, valid := activeReminders.nextReminderBefore(fireTime)
		if !valid {
			break
		}
		if fireTime.After(r.Deadline.Add(jitterWarningThreshold)) {
			logger.Warning("ProcessReminders: LATE by %v firing %v to %v:%v:%v", fireTime.Sub(r.Deadline), r.ID, r.Actor.Type, r.Actor.ID, r.Path)
		}

		logger.Debug("ProcessReminders: firing %v to %v:%v:%v (deadline %v)", r.ID, r.Actor.Type, r.Actor.ID, r.Path, r.Deadline)
		err := tell(ctx, r.Actor, r.Path, r.EncodedData, "application/json")
		if err != nil {
			logger.Error("ProcessReminders: firing %v raised error %v", r, err)
		}

		if r.Period > 0 {
			r.Deadline = fireTime.Add(r.Period)
			activeReminders.addReminder(r)
		}
	}

	arMutex.Unlock()
	logger.Debug("Completed reminder processing for time %v", fireTime)
}
