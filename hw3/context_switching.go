package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	// Configure to use only 1 OS thread
	runtime.GOMAXPROCS(1)
	fmt.Printf("GOMAXPROCS set to: %d\n\n", runtime.GOMAXPROCS(0))

	const rounds = 1_000_000

	// Create two unbuffered channels for ping-pong
	ping := make(chan struct{})
	pong := make(chan struct{})

	// Channel to signal completion
	done := make(chan struct{})

	// Goroutine 1: initiates the ping-pong
	go func() {
		for i := 0; i < rounds; i++ {
			ping <- struct{}{} // Send to goroutine 2
			<-pong             // Receive from goroutine 2
		}
		close(done) // Signal completion
	}()

	// Goroutine 2: responds to ping-pong
	go func() {
		for i := 0; i < rounds; i++ {
			<-ping             // Receive from goroutine 1
			pong <- struct{}{} // Send back to goroutine 1
		}
	}()

	// Start timing
	startTime := time.Now()

	// Wait for completion
	<-done

	elapsed := time.Since(startTime)

	// Calculate average switch time
	// Each round trip involves:
	//   1. G1 sends -> G2 receives (1 context switch)
	//   2. G2 sends -> G1 receives (1 context switch)
	// Total = 2 switches per round
	totalSwitches := 2 * rounds
	avgSwitchTime := elapsed / time.Duration(totalSwitches)

	fmt.Printf("Total rounds: %d\n", rounds)
	fmt.Printf("Total time: %v\n", elapsed)
	fmt.Printf("Total switches: %d\n", totalSwitches)
	fmt.Printf("Average switch time: %v\n", avgSwitchTime)
	fmt.Printf("Average switch time (ns): %.2f ns\n", float64(avgSwitchTime.Nanoseconds()))
}