package main

import (
	"flag"
	"fmt"
)

func main() {
	solutions := map[string]func(){
		"a": PartA, // Single-node kafka
		"b": PartB, // Multi-node kafka
	}

	part := flag.String("part", "b", "Specifies the part of the challenge to run. For example its value could be a, b, c.")
	flag.Parse()

	fn := solutions[*part]
	if fn == nil {
		fmt.Printf("Invalid part specified: '%s'. ", *part)
		fmt.Println("It could be that the solution is not yet ready for that part.")
		return
	}

	fn()
}
