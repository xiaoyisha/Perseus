package circuit

import (
	"Perseus/config"
	. "github.com/smartystreets/goconvey/convey"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"testing/quick"
	"time"
)

func TestGetCircuit(t *testing.T) {
	defer Flush()

	Convey("when calling GetCircuit", t, func() {
		var created bool
		var err error
		_, created, err = GetCircuitBreaker("foo")

		Convey("once, the circuit should be created", func() {
			So(err, ShouldBeNil)
			So(created, ShouldEqual, true)
		})

		Convey("twice, the circuit should be reused", func() {
			_, created, err = GetCircuitBreaker("foo")
			So(err, ShouldBeNil)
			So(created, ShouldEqual, false)
		})
	})
}

func TestMultithreadedGetCircuit(t *testing.T) {
	defer Flush()

	Convey("calling GetCircuit", t, func() {
		numThreads := 100
		var numCreates int32
		var numRunningRoutines int32
		var startingLine sync.WaitGroup
		var finishLine sync.WaitGroup
		startingLine.Add(1)
		finishLine.Add(numThreads)

		for i := 0; i < numThreads; i++ {
			go func() {
				if atomic.AddInt32(&numRunningRoutines, 1) == int32(numThreads) {
					startingLine.Done()
				} else {
					startingLine.Wait()
				}

				_, created, _ := GetCircuitBreaker("foo")

				if created {
					atomic.AddInt32(&numCreates, 1)
				}

				finishLine.Done()
			}()
		}

		finishLine.Wait()

		Convey("should be threadsafe", func() {
			So(numCreates, ShouldEqual, int32(1))
		})
	})
}

func TestReportEventMultiThreaded(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	run := func() bool {
		defer Flush()
		// Make the circuit easily open and close intermittently.
		config.ConfigureCommand("", config.CommandConfig{
			MaxConcurrentRequests:  1,
			ErrorPercentThreshold:  1,
			RequestVolumeThreshold: 1,
			SleepWindow:            10,
		})
		cb, _, _ := GetCircuitBreaker("")
		count := 5
		wg := &sync.WaitGroup{}
		wg.Add(count)
		c := make(chan bool, count)
		for i := 0; i < count; i++ {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						t.Error(r)
						c <- false
					} else {
						wg.Done()
					}
				}()
				// randomized eventType to open/close circuit
				eventType := "rejected"
				if rand.Intn(3) == 1 {
					eventType = "success"
				}
				err := cb.ReportEvent([]string{eventType}, time.Now(), time.Second)
				if err != nil {
					t.Error(err)
				}
				time.Sleep(time.Millisecond)
				// cb.IsOpen() internally calls cb.setOpen()
				cb.IsOpen()
			}()
		}
		go func() {
			wg.Wait()
			c <- true
		}()
		return <-c
	}
	if err := quick.Check(run, nil); err != nil {
		t.Error(err)
	}
}
