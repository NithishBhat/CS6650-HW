package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var m sync.Map
	var wg sync.WaitGroup
	
	numGoroutines := 50
	iterations := 1000

	start := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				// No manual locking needed
				m.Store(g*1000+i, i)
			}
		}(g)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Counting entries in sync.Map requires using Range
	length := 0
	m.Range(func(_, _ interface{}) bool {
		length++
		return true
	})

	fmt.Printf("Final map length: %d\n", length)
	fmt.Printf("Total time taken: %v\n", elapsed)
}