package runtime

import (
	"container/heap"
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/store"
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

// Reminder describes a time-triggered asynchronous invocation of a Path on an Actor
type Reminder struct {
	Actor       Actor
	ID          string        `json:"id"`
	key         string        // Implementation detail, do not serialize
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
	// Example: repeatingGreeter
	ID string `json:"id"`
	// The path to invoke on the actor instance when the reminder is fired
	// Example: sayHello
	Path string `json:"path"`
	// The time at which the reminder should first fire, specified as a string in an ISO-8601 compliant format
	Deadline time.Time `json:"deadline"`
	// The optional period parameter is a string encoding a GoLang Duration that is used to create a periodic reminder.
	// If a period is provided, then the reminder will be fired repeatedly by adding the period to the last fire time
	// to compute a new Deadline for the next invocation of the reminder.
	// Example: 30s
	Period string `json:"period,omitempty"`
	// An optional parameter containing an arbitray JSON value that will be provided as the
	// payload when the `path` is invoked on the actor instance.
	// Example: { msg: "Hello Friend!" }
	Data interface{} `json:"data,omitempty"`
}

// reminderParition returns the partition that is reponsible for all reminder processing for the argument actor.
// This assignment is stable.
func reminderParition(a Actor) int32 {
	// TODO: Implement a non-trivial yet stable assigment.
	//       when we do this, we must update rebalanceReminders
	return 0
}

// reminderKey returns a key suffix of the form: reminders_PARITION_ACTORTYPE_ACTORID_REMIDNERID
func reminderKey(a Actor, reminderID string) string {
	partition := strconv.Itoa(int(reminderParition(a)))
	return "reminders" + config.Separator + partition + config.Separator + a.Type + config.Separator + a.ID + config.Separator + reminderID
}

func persistReminder(r Reminder) {
	ts, _ := r.Deadline.MarshalText()
	rMap := make(map[string]string, 6)
	rMap["actorType"] = r.Actor.Type
	rMap["actorId"] = r.Actor.ID
	rMap["path"] = r.Path
	rMap["deadline"] = string(ts)
	if r.Period > 0 {
		rMap["period"] = r.Period.String()
	}
	if r.EncodedData != "" {
		rMap["encodedData"] = r.EncodedData
	}
	store.HSetMultiple(r.key, rMap)
}

func persistNewDeadline(key string, deadline time.Time) {
	ts, _ := deadline.MarshalText()
	store.HSet(key, "deadline", string(ts))
}

func loadReminder(rk string) (Reminder, error) {
	rMap, err := store.HGetAll(rk)
	if err != nil {
		return Reminder{}, err
	}
	logger.Debug("loadReminder: %v => %v", rk, rMap)
	var deadline time.Time
	err = deadline.UnmarshalText([]byte(rMap["deadline"]))
	if err != nil {
		return Reminder{}, err
	}
	var period time.Duration
	if ps, present := rMap["period"]; present {
		period, err = time.ParseDuration(ps)
		if err != nil {
			return Reminder{}, err
		}
	}
	r := Reminder{Actor: Actor{Type: rMap["actorType"], ID: rMap["actorId"]},
		key:         rk,
		Path:        rMap["path"],
		Deadline:    deadline,
		Period:      period,
		EncodedData: rMap["encodedData"],
	}
	return r, nil
}

// CancelReminders cancels all reminders that match the provided filter
func CancelReminders(actor Actor, payload string, contentType string, accepts string) (int, error) {
	var f reminderFilter
	if err := json.Unmarshal([]byte(payload), &f); err != nil {
		return 0, err
	}

	arMutex.Lock()
	found := activeReminders.cancelMatchingReminders(actor, f.ID)
	for _, cancelledReminder := range found {
		store.Del(cancelledReminder.key)
	}
	logger.Debug("Cancelled %v reminders matching (%v, %v)", found, actor, f.ID)
	arMutex.Unlock()

	return len(found), nil
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

// ScheduleReminder schedules a reminder
func ScheduleReminder(actor Actor, payload string, contentType string, accepts string) error {
	var data scheduleReminderPayload
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return err
	}
	rk := reminderKey(actor, data.ID)
	r := Reminder{Actor: actor, ID: data.ID, key: rk, Path: data.Path, Deadline: data.Deadline}
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

	arMutex.Lock()
	logger.Debug("ScheduleReminder: %v", r)
	activeReminders.cancelMatchingReminders(actor, r.ID) // FIXME: cancel is currently an O(# reminder) operation that is only needed if this reminder already exists.
	activeReminders.addReminder(r)
	persistReminder(r)
	arMutex.Unlock()

	return nil
}

// ProcessReminders causes all reminders with a deadline before fireTime to be scheduled for execution.
func processReminders(ctx context.Context, fireTime time.Time) {
	arMutex.Lock()
	logger.Debug("ProcessReminders: begin for time %v", fireTime)

	for {
		r, valid := activeReminders.nextReminderBefore(fireTime)
		if !valid {
			break
		}
		if fireTime.After(r.Deadline.Add(config.ActorReminderAcceptableDelay)) {
			logger.Warning("ProcessReminders: LATE by %v in firing %v to %v[%v]%v", fireTime.Sub(r.Deadline), r.ID, r.Actor.Type, r.Actor.ID, r.Path)
		}

		logger.Debug("ProcessReminders: firing %v to %v[%v]%v (deadline %v)", r.ID, r.Actor.Type, r.Actor.ID, r.Path, r.Deadline)
		err := TellActor(ctx, r.Actor, r.Path, r.EncodedData, "application/json")
		if err != nil {
			logger.Error("ProcessReminders: firing %v raised error %v", r, err)
		}

		if r.Period > 0 {
			r.Deadline = fireTime.Add(r.Period)
			activeReminders.addReminder(r)
			persistNewDeadline(r.key, r.Deadline)
		} else {
			store.Del(r.key)
		}
	}

	logger.Debug("ProcessReminders: completed for time %v", fireTime)
	arMutex.Unlock()
}

func containsZero(p []int32) bool {
	for _, v := range p {
		if v == 0 {
			return true
		}
	}
	return false
}

// rebalanceReminders is invoked asycnrhonously after a rebalancing operations to
// update this sidecar's reminderQueue to reflect the partitions it has been assigned
// by the rebalance operation.
func rebalanceReminders(ctx context.Context, priorPartitions []int32, newPartitions []int32) {
	prior := containsZero(priorPartitions)
	current := containsZero(newPartitions)

	// If nothing has changed, we can short-circuit without acquiring the mutex
	if prior == current {
		logger.Info("rebalanceReminders: responsibility unchanged (responsible = %v)", prior)
		return
	}

	// Assignments have changed; acquire the mutex and update data structures
	arMutex.Lock()
	logger.Info("rebalanceReminders: change in role: prior = %v current = %v", prior, current)

	// clear any prior assignment
	activeReminders = make(reminderQueue, 0)
	heap.Init(&activeReminders)

	if current {
		// Get the keys for all persisted reminders for this application
		rkeys, err := store.Keys("reminders" + config.Separator + "*")
		if err != nil {
			logger.Fatal("rebalanceReminders: Failure getting reminder keys: %v", err)
		}
		logger.Info("rebalanceReminders: found %v persisted reminders", len(rkeys))

		// For each key, load the persisted reminder and add to activeReminders
		for _, key := range rkeys {
			if r, err := loadReminder(key); err == nil {
				activeReminders.addReminder(r)
				logger.Info("scheduled persisted reminder %v", r)
			} else {
				logger.Error("rebalanceReminders: failed to schedule reminder with key %v due to %v", key, err)
			}
		}
	}

	logger.Info("rebalanceReminders: operation completed")
	arMutex.Unlock()
}
