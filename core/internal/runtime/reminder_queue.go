//
// Copyright IBM Corporation 2020,2022
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package runtime

import (
	"container/heap"
	"context"
	"net/http"
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

func (rq *reminderQueue) add(ctx context.Context, b binding) (int, error) {
	heap.Push(rq, &reminderEntry{r: b.(Reminder)})
	activeRemindersGauge.Inc()
	return http.StatusOK, nil
}

func (rq *reminderQueue) cancel(actor Actor, ID string) []binding {
	found := make([]binding, 0)
	for idx, elem := range *rq {
		if elem.r.Actor == actor && (ID == "" || elem.r.ID == ID) {
			(*rq)[idx].cancelled = true
			cancelledRemindersGuage.Inc()
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
		if re.cancelled {
			cancelledRemindersGuage.Dec()
		} else {
			activeRemindersGauge.Dec()
			return re.r, true
		}
	}
	return Reminder{}, false
}
