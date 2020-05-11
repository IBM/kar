package runtime

import (
	"container/heap"
	"context"
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
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
	TargetTime  time.Time     `json:"targetTime"`
	Period      time.Duration `json:"period,omitempty"` // 0 for one-shot reminders
	EncodedData string        `json:"encodedData,omitempty"`
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

// reminderKey returns a key suffix of the form: reminders_PARTITION_ACTORTYPE_ACTORID_REMINDERID
func reminderKey(a Actor, reminderID string, partition string) string {
	return "reminders" + config.Separator + partition + config.Separator + a.Type + config.Separator + a.ID + config.Separator + reminderID
}

func persistReminder(r Reminder) {
	ts, _ := r.TargetTime.MarshalText()
	rMap := make(map[string]string, 6)
	rMap["actorType"] = r.Actor.Type
	rMap["actorId"] = r.Actor.ID
	rMap["path"] = r.Path
	rMap["targetTime"] = string(ts)
	if r.Period > 0 {
		rMap["period"] = r.Period.String()
	}
	if r.EncodedData != "" {
		rMap["encodedData"] = r.EncodedData
	}
	store.HSetMultiple(r.key, rMap)
}

func persistTargetTime(key string, targetTime time.Time) {
	ts, _ := targetTime.MarshalText()
	store.HSet(key, "targetTime", string(ts))
}

func loadReminder(actor Actor, rk string) error {
	arMutex.Lock()
	found := activeReminders.findMatchingReminders(actor, strings.Split(rk, config.Separator)[4])
	if len(found) > 0 { // reminder is already loaded
		arMutex.Unlock()
		return nil
	}
	rMap, err := store.HGetAll(rk)
	if err != nil {
		arMutex.Unlock()
		return err
	}
	if len(rMap) == 0 { // reminder no longer exists
		arMutex.Unlock()
		return err
	}
	logger.Debug("loadReminder: %v => %v", rk, rMap)
	var targetTime time.Time
	err = targetTime.UnmarshalText([]byte(rMap["targetTime"]))
	if err != nil {
		arMutex.Unlock()
		return err
	}
	var period time.Duration
	if ps, present := rMap["period"]; present {
		period, err = time.ParseDuration(ps)
		if err != nil {
			arMutex.Unlock()
			return err
		}
	}
	r := Reminder{Actor: Actor{Type: rMap["actorType"], ID: rMap["actorId"]},
		key:         rk,
		Path:        rMap["path"],
		TargetTime:  targetTime,
		Period:      period,
		EncodedData: rMap["encodedData"],
	}
	activeReminders.addReminder(r)
	arMutex.Unlock()
	logger.Debug("scheduled persisted reminder %v", r)
	return nil
}

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

// CancelReminders cancels all reminders for actor that match reminderID ("" means match all)
func CancelReminders(actor Actor, reminderID string, contentType string, accepts string) int {
	arMutex.Lock()
	found := activeReminders.cancelMatchingReminders(actor, reminderID)
	for _, cancelledReminder := range found {
		store.Del(cancelledReminder.key)
	}
	logger.Debug("Cancelled %v reminders matching (%v, %v)", found, actor, reminderID)
	arMutex.Unlock()

	return len(found)
}

// GetReminders returns all reminders for actor that match reminderID ("" means match all)
func GetReminders(actor Actor, reminderID string, contentType string, accepts string) []Reminder {
	arMutex.Lock()
	found := activeReminders.findMatchingReminders(actor, reminderID)
	arMutex.Unlock()

	return found
}

// ScheduleReminder schedules a reminder
func ScheduleReminder(actor Actor, payload string, contentType string, accepts string) error {
	var data scheduleReminderPayload
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return err
	}
	r := Reminder{Actor: actor, ID: data.ID, Path: data.Path, TargetTime: data.TargetTime}
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
	keys, _ := store.Keys(reminderKey(actor, data.ID, "*"))
	if len(keys) > 0 { // reuse existing key
		r.key = keys[0]
	} else { // new key with random partition
		ps, _ := pubsub.Partitions()
		p := ps[rand.Int31n(int32(len(ps)))]
		r.key = reminderKey(actor, data.ID, strconv.Itoa(int(p)))
	}
	logger.Debug("ScheduleReminder: %v", r)
	activeReminders.cancelMatchingReminders(actor, r.ID) // FIXME: cancel is currently an O(# reminder) operation that is only needed if this reminder already exists.
	activeReminders.addReminder(r)
	persistReminder(r)
	arMutex.Unlock()

	return nil
}

// ProcessReminders causes all reminders with a targetTime before fireTime to be scheduled for execution.
func processReminders(ctx context.Context, fireTime time.Time) {
	arMutex.Lock()
	logger.Debug("ProcessReminders: begin for time %v", fireTime)

	for {
		r, valid := activeReminders.nextReminderBefore(fireTime)
		if !valid {
			break
		}
		if fireTime.After(r.TargetTime.Add(config.ActorReminderAcceptableDelay)) {
			logger.Warning("ProcessReminders: LATE by %v in firing %v to %v[%v]%v", fireTime.Sub(r.TargetTime), r.ID, r.Actor.Type, r.Actor.ID, r.Path)
		}

		logger.Debug("ProcessReminders: firing %v to %v[%v]%v (targetTime %v)", r.ID, r.Actor.Type, r.Actor.ID, r.Path, r.TargetTime)
		if err := TellActor(ctx, r.Actor, r.Path, r.EncodedData, "application/kar+json"); err != nil {
			logger.Debug("ProcessReminders: firing %v raised error %v", r, err)
			logger.Debug("ProcessReminders: ending this round; putting reminder back in queue to retry in next round")
			activeReminders.addReminder(r)
			break
		}

		if r.Period > 0 {
			r.TargetTime = fireTime.Add(r.Period)
			activeReminders.addReminder(r)
			persistTargetTime(r.key, r.TargetTime)
		} else {
			store.Del(r.key)
		}
	}

	logger.Debug("ProcessReminders: completed for time %v", fireTime)
	arMutex.Unlock()
}

// rebalanceReminders is invoked asynchronously after a rebalancing operations to
// scan reminders mapped to the partitions associated with this sidecar and
// ensure these reminders are loaded by their respective sidecars
func rebalanceReminders(ctx context.Context, partitions []int32) error {
	logger.Info("rebalanceReminders")

	for _, p := range partitions {
		// Get the keys for all persisted reminders for this partition
		rkeys, err := store.Keys("reminders" + config.Separator + strconv.Itoa(int(p)) + config.Separator + "*")
		if err != nil {
			return err
		}
		logger.Info("rebalanceReminders: found %v persisted reminders for partition %d", len(rkeys), p)

		// For each key, ping actor
		for _, key := range rkeys {
			parts := strings.Split(key, config.Separator)
			actor := Actor{Type: parts[2], ID: parts[3]}
			logger.Info("notifying %s", key)
			err := tellReminder(ctx, actor, key)
			if err != nil {
				if err != ctx.Err() {
					logger.Error("tell reminder %s failed: %v", key, err)
					logger.Error("rebalanceReminders: aborting")
				}
				return nil
			}
		}
	}

	logger.Info("rebalanceReminders: operation completed")

	return nil
}
