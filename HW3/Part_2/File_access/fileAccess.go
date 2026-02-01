package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

const iterations = 100000

func unbufferedWrite() time.Duration {
	
	f, err := os.Create("unbuffered.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	start := time.Now() 
	for i := 0; i < iterations; i++ {
	
		f.Write([]byte("experimental data line\n"))
	}

	return time.Since(start)
}

func bufferedWrite() time.Duration {

	f, err := os.Create("buffered.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()


	w := bufio.NewWriter(f)

	start := time.Now() 
	for i := 0; i < iterations; i++ {
	
		w.WriteString("experimental data line\n")
	}
	w.Flush() 
	
	return time.Since(start)
}

func main() {

	unbufDuration := unbufferedWrite()
	bufDuration := bufferedWrite()


	fmt.Printf("Unbuffered Duration: %v\n", unbufDuration)
	fmt.Printf("Buffered Duration:   %v\n", bufDuration)
}