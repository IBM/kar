package runtime

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
	activeReminders reminderQueue
	arMutex         sync.Mutex
)

func init() {
	activeReminders = make(reminderQueue, 0)
	heap.Init(&activeReminders)
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

type reminderFilter struct {
	// An optional reminder ID
	ID string `json:"id,omitempty"`
}

// ScheduleReminderPayload is the JSON request body for scheduling a new reminder
type scheduleReminderPayload struct {
	// The ID to use for this reminder
	ID string `json:"id"`
	// The path to invoke on the actor instance when the reminder is fired
	Path string `json:"path"`
	// The time at which the reminder should first fire, specified as a string in an ISO-8601 compliant format
	Deadline time.Time `json:"deadline"`
	// The optional period parameter is a string encoding a GoLang Duration that is used to create a periodic reminder.
	// If a period is provided, then the reminder will be fired repeatedly by adding the period to the last fire time
	// to compute a new Deadline for the next invocation of the reminder.
	Period string `json:"period,omitempty"`
	// An optional parameter containing an arbitray JSON value that will be provided as the
	// payload when the `path` is invoked on the actor instance.
	Data interface{} `json:"data,omitempty"`
}

// CancelReminders cancels all reminders that match the provided filter
func CancelReminders(actor Actor, payload string, contentType string, accepts string) (int, error) {
	var f reminderFilter
	if err := json.Unmarshal([]byte(payload), &f); err != nil {
		return 0, err
	}

	arMutex.Lock()
	found := activeReminders.cancelMatchingReminders(actor, f.ID)
	arMutex.Unlock()

	logger.Debug("Cancelled %v reminders matching (%v, %v)", found, actor, f.ID)
	return found, nil
}

// GetReminders returns all reminders that match the provided filter
func GetReminders(actor Actor, payload string, contentType string, accepts string) ([]Reminder, error) {
	var f reminderFilter
	if err := json.Unmarshal([]byte(payload), &f); err != nil {
		return nil, err
	}

	arMutex.Lock()
	found := activeReminders.findMatchingReminders(actor, f.ID)
	arMutex.Unlock()

	return found, nil
}

// ScheduleReminder schedules a new reminder
func ScheduleReminder(actor Actor, payload string, contentType string, accepts string) error {
	var data scheduleReminderPayload
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return err
	}

	r := Reminder{Actor: actor, Path: data.Path, ID: data.ID, Deadline: data.Deadline}
	if data.Period != "" {
		period, err := time.ParseDuration(data.Period)
		if err != nil {
			return err
		}
		r.Period = period
	}
	if data.Data != nil {
		buf, err := json.Marshal(data.Data)
		if err != nil {
			return err
		}
		r.EncodedData = string(buf)
	}

	logger.Debug("ScheduleReminder: %v", r)
	arMutex.Lock()
	activeReminders.addReminder(r)
	arMutex.Unlock()

	return nil
}

// ProcessReminders causes all reminders with a deadline before fireTime to be scheduled for execution.
func processReminders(ctx context.Context, fireTime time.Time) {
	logger.Debug("ProcessReminders invoked at %v", fireTime)
	arMutex.Lock()

	for {
		r, valid := activeReminders.nextReminderBefore(fireTime)
		if !valid {
			break
		}
		if fireTime.After(r.Deadline.Add(config.ActorReminderAcceptableDelay)) {
			logger.Warning("ProcessReminders: LATE by %v firing %v to %v:%v:%v", fireTime.Sub(r.Deadline), r.ID, r.Actor.Type, r.Actor.ID, r.Path)
		}

		logger.Debug("ProcessReminders: firing %v to %v:%v:%v (deadline %v)", r.ID, r.Actor.Type, r.Actor.ID, r.Path, r.Deadline)
		err := TellActor(ctx, r.Actor, r.Path, r.EncodedData, "application/json")
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
