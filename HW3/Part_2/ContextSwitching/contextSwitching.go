package main

import (
	"fmt"
	"runtime"
	"time"
)

func pingPong(iterations int) time.Duration {
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})
	done := make(chan struct{})

	start := time.Now()

	// Goroutine 1: Wait then Send
	go func() {
		for i := 0; i < iterations; i++ {
			<-ch1
			ch2 <- struct{}{}
		}
	}()

	// Goroutine 2: Send then Wait
	go func() {
		for i := 0; i < iterations; i++ {
			ch1 <- struct{}{}
			<-ch2
		}
		done <- struct{}{}
	}()

	<-done
	return time.Since(start)
}

func main() {
	const count = 1000000

	// 1. Single OS Thread restriction
	runtime.GOMAXPROCS(1) 
	dur1 := pingPong(count)
	avg1 := float64(dur1.Nanoseconds()) / (2 * float64(count)) // 

	// 2. Multi-threaded (remove restriction)
	runtime.GOMAXPROCS(runtime.NumCPU()) 
	dur2 := pingPong(count)
	avg2 := float64(dur2.Nanoseconds()) / (2 * float64(count))

	fmt.Printf("--- GOMAXPROCS(1) ---\n")
	fmt.Printf("Total Duration: %v\n", dur1)
	fmt.Printf("Average Switch Time: %.2f ns\n\n", avg1)

	fmt.Printf("--- Multi-threaded ---\n")
	fmt.Printf("Total Duration: %v\n", dur2)
	fmt.Printf("Average Switch Time: %.2f ns\n", avg2)
}