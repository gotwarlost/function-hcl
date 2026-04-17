// Package queue provides a simple queue implementation with support for
// deduping equivalent jobs using the notion of a unique key for each job.
package queue

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/utils/perf"
)

// Key is a unique key for a job.
type Key string

func (k Key) String() string {
	return string(k)
}

type job struct {
	id       uint64
	key      Key
	fn       func() error
	l        sync.Mutex
	canceled bool
}

func (j *job) setCanceled(flag bool) {
	j.l.Lock()
	defer j.l.Unlock()
	j.canceled = flag
}

func (j *job) isCanceled() bool {
	j.l.Lock()
	defer j.l.Unlock()
	return j.canceled
}

type waiter struct {
	ids map[uint64]struct{}
	ch  chan struct{}
}

// Queue implements the queue.
type Queue struct {
	l                sync.Mutex
	concurrency      int
	nextID           uint64
	work             chan *job
	pendingJobsByKey map[Key]*job
	runningJobsByKey map[Key]*job
	waiters          []waiter
}

// New creates a queue with the supplied concurrency.
func New(concurrency int) *Queue {
	return &Queue{
		concurrency:      concurrency,
		work:             make(chan *job, 1000),
		pendingJobsByKey: map[Key]*job{},
		runningJobsByKey: map[Key]*job{},
	}
}

// Start starts the background processing for the queue.
// Processing stops when the context is canceled.
func (q *Queue) Start(ctx context.Context) {
	processItem := func(j *job) {
		defer perf.Measure("job: " + j.key.String())()
		q.setProcessing(j)
		defer q.setDone(j)
		if !j.isCanceled() {
			err := j.fn()
			if err != nil {
				log.Printf("Error in job %s [%d]: %v", j.key, j.id, err)
			}
		}
	}
	for i := 0; i < q.concurrency; i++ {
		go func() {
			for {
				select {
				case j := <-q.work:
					processItem(j)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

func (q *Queue) setProcessing(j *job) {
	q.l.Lock()
	defer q.l.Unlock()
	delete(q.pendingJobsByKey, j.key)
	q.runningJobsByKey[j.key] = j
}

func (q *Queue) setDone(j *job) {
	q.l.Lock()
	defer q.l.Unlock()
	delete(q.runningJobsByKey, j.key)
	newWaiters := make([]waiter, 0, len(q.waiters))
	for _, w := range q.waiters {
		delete(w.ids, j.id)
		if len(w.ids) == 0 {
			close(w.ch)
			continue
		}
		newWaiters = append(newWaiters, w)
	}
	q.waiters = newWaiters
}

// Enqueue enqueues a job with a specific key. The job will not be added to the queue
// if there already exists a job that is waiting to be processed which has the same key.
// Note that a running job for the same key will still cause this one to be enqueued.
func (q *Queue) Enqueue(key Key, fn func() error) {
	q.l.Lock()
	defer q.l.Unlock()
	j := q.pendingJobsByKey[key]
	if j != nil {
		j.setCanceled(false)
		return
	}
	j = &job{
		id:  atomic.AddUint64(&q.nextID, 1),
		key: key,
		fn:  fn,
	}
	q.pendingJobsByKey[j.key] = j
	q.work <- j
}

// Dequeue removes any waiting job having the supplied key. Running jobs are not affected.
func (q *Queue) Dequeue(key Key) {
	q.l.Lock()
	defer q.l.Unlock()
	j := q.pendingJobsByKey[key]
	if j != nil {
		j.setCanceled(true)
	}
}

// WaitForKey waits until the running and waiting jobs for this key have completed.
// Note that it will not wait forever even the queue keeps getting new job with the same key
// after this function is called.
func (q *Queue) WaitForKey(key Key) {
	q.l.Lock()
	j1 := q.runningJobsByKey[key]
	j2 := q.pendingJobsByKey[key]
	if j1 == nil && j2 == nil {
		q.l.Unlock()
		return
	}
	ch := q.addWaiter(j1, j2)
	q.l.Unlock()
	<-ch
}

func (q *Queue) addWaiter(j1, j2 *job) <-chan struct{} {
	ch := make(chan struct{})
	m := map[uint64]struct{}{}
	if j1 != nil {
		m[j1.id] = struct{}{}
	}
	if j2 != nil {
		m[j2.id] = struct{}{}
	}
	w := waiter{
		ids: m,
		ch:  ch,
	}
	q.waiters = append(q.waiters, w)
	return ch
}
