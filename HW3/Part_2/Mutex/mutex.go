package main

import (
	"fmt"
	"sync"
	"time"
)

type Container struct {
	mu       sync.Mutex
	counters map[int]int
}

func (c *Container) inc(key int, val int) {
	c.mu.Lock()         // Lock before write
	defer c.mu.Unlock() // Unlock after write
	c.counters[key] = val
}

func main() {
	c := Container{
		counters: make(map[int]int),
	}

	var wg sync.WaitGroup
	start := time.Now()

	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				// Writing 1,000 distinct key/value pairs per goroutine
				c.inc(g*1000+i, i)
			}
		}(g)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Println("Final map length:", len(c.counters))
	fmt.Println("Total time taken:", elapsed)
}