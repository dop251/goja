package goja

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestProfiler(t *testing.T) {

	err := StartProfile(nil)
	if err != nil {
		t.Fatal(err)
	}

	vm := New()
	go func() {
		_, err := vm.RunScript("test123.js", `
			const a = 2 + 2;
			function loop() {
				for(;;) {}
			}
			loop();
		`)
		if err != nil {
			if _, ok := err.(*InterruptedError); !ok {
				panic(err)
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)

	atomic.StoreInt32(&globalProfiler.enabled, 0)
	pr := globalProfiler.p.stop()

	if len(pr.Sample) == 0 {
		t.Fatal("No samples were recorded")
	}

	var running bool
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		globalProfiler.p.mu.Lock()
		running = globalProfiler.p.running
		globalProfiler.p.mu.Unlock()
		if !running {
			break
		}
	}
	if running {
		t.Fatal("The profiler is still running")
	}
	vm.Interrupt(nil)
}
