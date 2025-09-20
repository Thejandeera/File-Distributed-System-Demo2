package time_sync

import (
	"fmt"
	"sync"
	"time"
)

// LamportClock represents a simple logical clock.
type LamportClock struct {
	mu    sync.Mutex
	clock int
}

// NewLamportClock creates and returns a new LamportClock.
func NewLamportClock() *LamportClock {
	return &LamportClock{clock: 0}
}

// Tick simulates a local event, incrementing the clock.
func (lc *LamportClock) Tick() int {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.clock++
	return lc.clock
}

// Receive processes a received timestamp and updates the clock.
func (lc *LamportClock) Receive(received int) int {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if received > lc.clock {
		lc.clock = received
	}
	lc.clock++
	return lc.clock
}

// Value returns the current clock value.
func (lc *LamportClock) Value() int {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.clock
}

// SimulateLogicalClocks runs 2 clocks and shows tick & receive
func SimulateLogicalClocks() {
	A := NewLamportClock()
	B := NewLamportClock()

	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(1 * time.Second)
			val := A.Tick()
			fmt.Printf("⏱ Clock A ticked to %d\n", val)

			if i == 2 {
				// Simulate sending A's clock value to B
				newVal := B.Receive(val)
				fmt.Printf("📨 Clock B received from A and updated to %d\n", newVal)
			}
		}
	}()

	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(1200 * time.Millisecond)
			val := B.Tick()
			fmt.Printf("⏱ Clock B ticked to %d\n", val)
		}
	}()
}
