package eventloop

import (
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja/nodejs/console"
	"github.com/dop251/goja/nodejs/require"
)

type job struct {
	goja.Callable
	args      []goja.Value
	cancelled bool
}

type timer struct {
	job
	timer *time.Timer
}

type interval struct {
	job
	ticker   *time.Ticker
	stopChan chan struct{}
}

type EventLoop struct {
	vm       *goja.Runtime
	jobChan  chan func()
	jobCount int32
	running  bool
}

func NewEventLoop() *EventLoop {
	vm := goja.New()

	loop := &EventLoop{
		vm:      vm,
		jobChan: make(chan func()),
	}

	new(require.Registry).Enable(vm)
	console.Enable(vm)
	vm.Set("setTimeout", loop.setTimeout)
	vm.Set("setInterval", loop.setInterval)
	vm.Set("clearTimeout", loop.clearTimeout)
	vm.Set("clearInterval", loop.clearInterval)

	return loop
}

func (loop *EventLoop) schedule(call goja.FunctionCall, repeating bool) goja.Value {
	if fn, ok := goja.AssertFunction(call.Argument(0)); ok {
		delay := call.Argument(1).ToInteger()
		var args []goja.Value
		if len(call.Arguments) > 2 {
			args = call.Arguments[2:]
		}
		if repeating {
			return loop.vm.ToValue(loop.addInterval(fn, time.Duration(delay)*time.Millisecond, args))
		} else {
			return loop.vm.ToValue(loop.addTimeout(fn, time.Duration(delay)*time.Millisecond, args))
		}
	}
	return nil
}

func (loop *EventLoop) setTimeout(call goja.FunctionCall) goja.Value {
	return loop.schedule(call, false)
}

func (loop *EventLoop) setInterval(call goja.FunctionCall) goja.Value {
	return loop.schedule(call, true)
}

// Run calls the specified function, starts the event loop and waits until there are no more delayed jobs to run
// after which it stops the loop and returns.
// The instance of goja.Runtime that is passed to the function and any Values derived from it must not be used outside
// of the function.
// Do NOT use this function while the loop is already running. Use RunOnLoop() instead.
func (loop *EventLoop) Run(fn func(*goja.Runtime)) {
	fn(loop.vm)
	loop.run()
}

// Start the event loop in the background. The loop continues to run until Stop() is called.
func (loop *EventLoop) Start() {
	go loop.runInBackground()
}

// Stop the loop that was started with Start(). After this function returns there will be no more jobs executed
// by the loop. It is possible to call Start() or Run() again after this to resume the execution.
// Note, it does not cancel active timeouts.
func (loop *EventLoop) Stop() {
	ch := make(chan struct{})

	loop.jobChan <- func() {
		loop.running = false
		ch <- struct{}{}
	}

	<-ch
}

// RunOnLoop schedules to run the specified function in the context of the loop as soon as possible.
// The order of the runs is preserved (i.e. the functions will be called in the same order as calls to RunOnLoop())
// The instance of goja.Runtime that is passed to the function and any Values derived from it must not be used outside
// of the function.
func (loop *EventLoop) RunOnLoop(fn func(*goja.Runtime)) {
	loop.jobChan <- func() {
		fn(loop.vm)
	}
}

func (loop *EventLoop) run() {
	loop.running = true
	for loop.running && loop.jobCount > 0 {
		job, ok := <-loop.jobChan
		if !ok {
			break
		}
		job()
	}
}

func (loop *EventLoop) runInBackground() {
	loop.running = true
	for job := range loop.jobChan {
		job()
		if !loop.running {
			break
		}
	}
}

func (loop *EventLoop) addTimeout(f goja.Callable, timeout time.Duration, args []goja.Value) *timer {
	t := &timer{
		job: job{Callable: f, args: args},
	}

	t.timer = time.AfterFunc(timeout, func() {
		loop.jobChan <- func() {
			loop.doTimeout(t)
		}
	})

	loop.jobCount++
	return t
}

func (loop *EventLoop) addInterval(f goja.Callable, timeout time.Duration, args []goja.Value) *interval {
	i := &interval{
		job:      job{Callable: f, args: args},
		ticker:   time.NewTicker(timeout),
		stopChan: make(chan struct{}),
	}

	go i.run(loop)
	loop.jobCount++
	return i
}

func (loop *EventLoop) doTimeout(t *timer) {
	if !t.cancelled {
		t.Callable(nil, t.args...)
		t.cancelled = true
		loop.jobCount--
	}
}

func (loop *EventLoop) doInterval(i *interval) {
	if !i.cancelled {
		i.Callable(nil, i.args...)
	}
}

func (loop *EventLoop) clearTimeout(t *timer) {
	if !t.cancelled {
		t.timer.Stop()
		t.cancelled = true
		loop.jobCount--
	}
}

func (loop *EventLoop) clearInterval(i *interval) {
	if !i.cancelled {
		i.cancelled = true
		i.stopChan <- struct{}{}
		loop.jobCount--
	}
}

func (i *interval) run(loop *EventLoop) {
	for {
		select {
		case <-i.stopChan:
			i.ticker.Stop()
			break
		case <-i.ticker.C:
			loop.jobChan <- func() {
				loop.doInterval(i)
			}
		}
	}
}
