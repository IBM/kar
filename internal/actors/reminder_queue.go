package actors

import (
	"container/heap"
	"time"
)

type reminder struct {
	deadline time.Time
	period   time.Duration // 0 for one-shot reminders
	id       string        // TODO: make this an actor type, id, method
}

type reminderEntry struct {
	r     reminder
	index int
}

type reminderQueue []*reminderEntry

func (rq reminderQueue) Len() int { return len(rq) }

func (rq reminderQueue) Less(i, j int) bool {
	// Deadlines further in the future have lower priority
	return rq[i].r.deadline.Before(rq[j].r.deadline)
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

func (rq *reminderQueue) addReminder(r reminder) {
	heap.Push(rq, &reminderEntry{r: r})
}

func (rq *reminderQueue) nextReminderBefore(t time.Time) (reminder, bool) {
	if len(*rq) > 0 && (*rq)[0].r.deadline.Before(t) {
		re := heap.Pop(rq)
		return re.(*reminderEntry).r, true
	}
	return reminder{}, false
}
