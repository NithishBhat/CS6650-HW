package main

import (
	"fmt"

)

func main() {
	var ops uint64

	for range 50 {
		go func() {
			for range 1000 {
				ops++ 
			}
		}()
	}


	fmt.Println( ops)
}