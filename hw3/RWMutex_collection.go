package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeMapRW wraps a map with an RWMutex for thread-safe access
type SafeMapRW struct {
	mu sync.RWMutex
	m  map[int]int
}

// Set writes a key-value pair to the map
func (sm *SafeMapRW) Set(key, value int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] = value
}

// Len returns the number of entries in the map
func (sm *SafeMapRW) Len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.m)
}

func main() {
	// Create our thread-safe map with RWMutex
	safeMap := SafeMapRW{
		m: make(map[int]int),
	}

	var wg sync.WaitGroup

	// Start timing
	start := time.Now()

	// Spawn 50 goroutines
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			// Each goroutine writes 1,000 entries
			for i := 0; i < 1000; i++ {
				safeMap.Set(g*1000+i, i)
			}
		}(g)
	}

	wg.Wait()

	// Stop timing
	elapsed := time.Since(start)

	// Print results
	fmt.Println("len(m):", safeMap.Len())
	fmt.Printf("Time taken: %v\n", elapsed)
}
