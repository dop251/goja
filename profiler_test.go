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
		time.Sleep(10 * time.Millisecond)
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

func TestProfiler1(t *testing.T) {
	t.Skip("This test takes too long with race detector enabled and is non-deterministic. It's left here mostly for documentation purposes.")

	err := StartProfile(nil)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		sleep := func() {
			time.Sleep(1 * time.Second)
		}
		// Spawn new vms faster than once every 10ms (the profiler interval) and make sure they don't finish too soon.
		// This means (*profiler).run() won't be fast enough to collect the samples, so they must be collected
		// after the profiler is stopped.
		for i := 0; i < 500; i++ {
			go func() {
				vm := New()
				vm.Set("sleep", sleep)
				_, err := vm.RunScript("test123.js", `
				function loop() {
					for (let i = 0; i < 50000; i++) {
						const a = Math.pow(Math.Pi, Math.Pi);
					}
				}
				loop();
				sleep();
				`)
				if err != nil {
					if _, ok := err.(*InterruptedError); !ok {
						panic(err)
					}
				}
			}()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	atomic.StoreInt32(&globalProfiler.enabled, 0)
	pr := globalProfiler.p.stop()

	if len(pr.Sample) == 0 {
		t.Fatal("No samples were recorded")
	}
}
