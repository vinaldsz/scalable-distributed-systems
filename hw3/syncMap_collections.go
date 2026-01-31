package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
    var m sync.Map

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
                m.Store(g*1000+i, i)
            }
        }(g)
    }

    wg.Wait()

    // Stop timing
    elapsed := time.Since(start)

    // Count entries using Range
    count := 0
    m.Range(func(key, value interface{}) bool {
        count++
        return true
    })

    // Print results
    fmt.Println("len(m):", count)
    fmt.Printf("Time taken: %v\n", elapsed)
}
