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

// ProcessReminders causes all reminders with a deadline before time to be scheduled for execution.
func ProcessReminders(ctx context.Context, time time.Time) {
	logger.Debug("ProcessReminders invoked at %v", time)
	arMutex.Lock()

	for {
		r, valid := activeReminders.nextReminderBefore(time)
		if !valid {
			break
		}
		logger.Info("at %v scheduling %v (deadline %v)", time, r.id, r.deadline)
		if r.period > 0 {
			r.deadline = r.deadline.Add(r.period)
			activeReminders.addReminder(r)
		}
	}

	arMutex.Unlock()
	logger.Debug("Completed reminder processing for time %v", time)
}

// ScheduleOneShotReminder schedules a reminder
func ScheduleOneShotReminder(id string, deadline time.Time) {
	r := reminder{deadline: deadline, id: id}
	arMutex.Lock()
	activeReminders.addReminder(r)
	arMutex.Unlock()
}

// SchedulePeriodicReminder schedules a reminder
func SchedulePeriodicReminder(id string, deadline time.Time, period time.Duration) {
	r := reminder{deadline: deadline, period: period, id: id}
	arMutex.Lock()
	activeReminders.addReminder(r)
	arMutex.Unlock()
}
