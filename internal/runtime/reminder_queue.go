package runtime

import (
	"container/heap"
	"context"
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
	// targetTimes further in the future have lower priority
	return rq[i].r.TargetTime.Before(rq[j].r.TargetTime)
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

func (rq *reminderQueue) add(ctx context.Context, b binding) error {
	heap.Push(rq, &reminderEntry{r: b.(Reminder)})
	return nil
}

func (rq *reminderQueue) cancel(actor Actor, ID string) []binding {
	found := make([]binding, 0)
	for idx, elem := range *rq {
		if elem.r.Actor == actor && (ID == "" || elem.r.ID == ID) {
			(*rq)[idx].cancelled = true
			found = append(found, (*rq)[idx].r)
		}
	}
	return found
}

func (rq *reminderQueue) find(actor Actor, ID string) []binding {
	result := make([]binding, 0)
	for _, elem := range *rq {
		if elem.r.Actor == actor && (ID == "" || elem.r.ID == ID) && !elem.cancelled {
			result = append(result, elem.r)
		}
	}
	return result
}

func (rq *reminderQueue) nextReminderBefore(t time.Time) (Reminder, bool) {
	for len(*rq) > 0 && (*rq)[0].r.TargetTime.Before(t) {
		re := heap.Pop(rq).(*reminderEntry)
		if !re.cancelled {
			return re.r, true
		}
	}
	return Reminder{}, false
}
