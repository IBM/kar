package actors

import (
	"container/heap"
	"time"
)

type reminderEntry struct {
	r         Reminder
	cancelled bool
	index     int
}

type reminderQueue []*reminderEntry

func (rq reminderQueue) Len() int { return len(rq) }

func (rq reminderQueue) Less(i, j int) bool {
	// Deadlines further in the future have lower priority
	return rq[i].r.Deadline.Before(rq[j].r.Deadline)
}

func (rq reminderQueue) Swap(i, j int) {
	rq[i], rq[j] = rq[j], rq[i]
	rq[i].index = i
	rq[j].index = j
}

func (rq *reminderQueue) Push(x interface{}) {
	n := len(*rq)
	r := x.(*reminderEntry)
	r.index = n
	*rq = append(*rq, r)
}

func (rq *reminderQueue) Pop() interface{} {
	old := *rq
	n := len(old)
	r := old[n-1]
	r.index = -1
	*rq = old[0 : n-1]
	return r
}

func (rq *reminderQueue) addReminder(r Reminder) {
	heap.Push(rq, &reminderEntry{r: r})
}

func (rq *reminderQueue) cancelReminder(actor Actor, ID string) bool {
	found := false
	for idx, elem := range *rq {
		if elem.r.Actor == actor && (ID == "" || elem.r.ID == ID) {
			(*rq)[idx].cancelled = true
			found = true
		}
	}
	return found
}

func (rq *reminderQueue) findMatchingReminders(actor Actor, ID string) []Reminder {
	result := make([]Reminder, 0)
	for _, elem := range *rq {
		if elem.r.Actor == actor && (ID == "" || elem.r.ID == ID) {
			result = append(result, elem.r)
		}
	}
	return result
}

func (rq *reminderQueue) nextReminderBefore(t time.Time) (Reminder, bool) {
	for len(*rq) > 0 && (*rq)[0].r.Deadline.Before(t) {
		re := heap.Pop(rq).(*reminderEntry)
		if !re.cancelled {
			return re.r, true
		}
	}
	return Reminder{}, false
}
