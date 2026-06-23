package goja

import (
	"os"
	"testing"
)

// externalJobRunner is a host-owned FIFO queue for promise jobs captured
// via PromiseJobEnqueuer and drained through RunPromiseJob.
type externalJobRunner struct {
	queue []func() // FIFO microtask queue
	vm    *Runtime
}

// enqueue implements PromiseJobEnqueuer; appends to the queue tail.
func (r *externalJobRunner) enqueue(job func()) {
	r.queue = append(r.queue, job)
}

// drain runs each queued job via RunPromiseJob (which re-enqueues triggered
// jobs). On an uncatchable error the queue is cleared.
func (r *externalJobRunner) drain() error {
	for len(r.queue) > 0 {
		job := r.queue[0]
		r.queue = r.queue[1:]
		if err := r.vm.RunPromiseJob(job); err != nil {
			r.queue = nil
			return err
		}
	}
	return nil
}

// TestTC39ExternalPromiseJobs runs test262 with a host-owned job queue:
// jobs captured by PromiseJobEnqueuer are drained via RunPromiseJob after
// each RunProgram.
func TestTC39ExternalPromiseJobs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	if _, err := os.Stat(tc39BASE); err != nil {
		t.Skipf("If you want to run tc39 tests, download them from https://github.com/tc39/test262 and put into %s. See .tc39_test262_checkout.sh for the latest working commit id. (%v)", tc39BASE, err)
	}

	ctx := &tc39TestCtx{
		base:                tc39BASE,
		externalPromiseJobs: true,
	}
	ctx.init()

	t.Run("tc39-external", func(t *testing.T) {
		ctx.t = t
		ctx.runAllTC39Tests()
		ctx.flush()
	})
}
