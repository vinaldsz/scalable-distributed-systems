package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeMap wraps a map with a mutex for thread-safe access
type SafeMap struct {
	mu sync.Mutex
	m  map[int]int
}

// Set writes a key-value pair to the map
func (sm *SafeMap) Set(key, value int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()  // runs at the end
	sm.m[key] = value
}

// Len returns the number of entries in the map
func (sm *SafeMap) Len() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return len(sm.m)
}

func main() {
	// Create our thread-safe map
	safeMap := SafeMap{
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