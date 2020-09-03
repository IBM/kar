package runtime

import (
	"container/heap"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/store"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	activeReminders = &reminderQueue{}
	arMutex         = &sync.Mutex{}
)

func init() {
	heap.Init(activeReminders)
	pairs["reminders"] = pair{bindings: activeReminders, mu: arMutex}
}

// Reminder describes a time-triggered asynchronous invocation of a Path on an Actor
type Reminder struct {
	Actor       Actor
	ID          string        `json:"id"`
	key         string        // Implementation detail, do not serialize
	Path        string        `json:"path"`
	TargetTime  time.Time     `json:"targetTime"`
	Period      time.Duration `json:"period,omitempty"` // 0 for one-shot reminders
	EncodedData string        `json:"encodedData,omitempty"`
}

func (r Reminder) k() string {
	return r.key
}

// ScheduleReminderPayload is the JSON request body for scheduling a new reminder
type scheduleReminderPayload struct {
	// The path to invoke on the actor instance when the reminder is fired
	// Example: sayHello
	Path string `json:"path"`
	// The time at which the reminder should first fire, specified as a string in an ISO-8601 compliant format
	TargetTime time.Time `json:"targetTime"`
	// The optional period parameter is a string encoding a GoLang Duration that is used to create a periodic reminder.
	// If a period is provided, then the reminder will be fired repeatedly by adding the period to the last fire time
	// to compute a new TargetTime for the next invocation of the reminder.
	// Example: 30s
	Period string `json:"period,omitempty"`
	// An optional parameter containing an arbitrary JSON value that will be provided as the
	// payload when the `path` is invoked on the actor instance.
	// Example: { msg: "Hello Friend!" }
	Data interface{} `json:"data,omitempty"`
}

func persistReminder(r Reminder) map[string]string {
	ts, _ := r.TargetTime.MarshalText()
	rMap := make(map[string]string, 6)
	rMap["actorType"] = r.Actor.Type
	rMap["actorId"] = r.Actor.ID
	rMap["id"] = r.ID
	rMap["path"] = r.Path
	rMap["targetTime"] = string(ts)
	if r.Period > 0 {
		rMap["period"] = r.Period.String()
	}
	if r.EncodedData != "" {
		rMap["encodedData"] = r.EncodedData
	}
	return rMap
}

func persistTargetTime(key string, targetTime time.Time) {
	ts, _ := targetTime.MarshalText()
	store.HSet(key, "targetTime", string(ts))
}

func (rq *reminderQueue) load(actor Actor, id, key string, rMap map[string]string) (binding, error) {
	var targetTime time.Time
	err := targetTime.UnmarshalText([]byte(rMap["targetTime"]))
	if err != nil {
		return nil, err
	}
	var period time.Duration
	if ps, present := rMap["period"]; present {
		period, err = time.ParseDuration(ps)
		if err != nil {
			return nil, err
		}
	}
	r := Reminder{Actor: Actor{Type: rMap["actorType"], ID: rMap["actorId"]},
		ID:          rMap["id"],
		key:         key,
		Path:        rMap["path"],
		TargetTime:  targetTime,
		Period:      period,
		EncodedData: rMap["encodedData"],
	}
	return r, nil
}

func (rq *reminderQueue) parse(actor Actor, id, key, payload string) (binding, map[string]string, error) {
	var data scheduleReminderPayload
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil, nil, err
	}
	r := Reminder{Actor: actor, ID: id, key: key, Path: data.Path, TargetTime: data.TargetTime}
	if data.Period != "" {
		period, err := time.ParseDuration(data.Period)
		if err != nil {
			return nil, nil, err
		}
		r.Period = period
	}
	if data.Data != nil {
		buf, err := json.Marshal(data.Data)
		if err != nil {
			return nil, nil, err
		}
		r.EncodedData = string(buf)
	}
	return r, persistReminder(r), nil
}

/* TODO
func migrateReminders(ctx context.Context, actor Actor) {
	arMutex.Lock()
	found := activeReminders.cancelMatchingReminders(actor, "")
	arMutex.Unlock()
	for _, r := range found {
		err := tellReminder(ctx, r.Actor, r.key)
		if err != nil {
			if err != ctx.Err() {
				logger.Error("tell reminder %s failed: %v", r.key, err)
			}
			break
		}
	}
}
*/

// ProcessReminders causes all reminders with a targetTime before fireTime to be scheduled for execution.
func processReminders(ctx context.Context, fireTime time.Time) {
	arMutex.Lock()

	for {
		r, valid := activeReminders.nextReminderBefore(fireTime)
		if !valid {
			break
		}
		if fireTime.After(r.TargetTime.Add(config.ActorReminderAcceptableDelay)) {
			logger.Warning("ProcessReminders: LATE by %v in firing %v to %v[%v]%v", fireTime.Sub(r.TargetTime), r.ID, r.Actor.Type, r.Actor.ID, r.Path)
		}

		logger.Debug("ProcessReminders: firing %v to %v[%v]%v (targetTime %v)", r.ID, r.Actor.Type, r.Actor.ID, r.Path, r.TargetTime)
		if err := TellActor(ctx, r.Actor, r.Path, r.EncodedData, "application/kar+json", "POST", false); err != nil {
			logger.Debug("ProcessReminders: firing %v raised error %v", r, err)
			logger.Debug("ProcessReminders: ending this round; putting reminder back in queue to retry in next round")
			activeReminders.add(ctx, r)
			break
		}

		if r.Period > 0 {
			r.TargetTime = fireTime.Add(r.Period)
			activeReminders.add(ctx, r)
			persistTargetTime(r.key, r.TargetTime)
		} else {
			store.Del(r.key)
		}
	}

	arMutex.Unlock()
}
