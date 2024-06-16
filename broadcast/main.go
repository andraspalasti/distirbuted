package main

import (
	"flag"
	"fmt"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	solutions := map[string]func(){
		"a": PartA, // Single node broadcast
		"b": PartB, // Multi node broadcast
		"c": PartC, // Fault tolerant broadcast
		"d": PartD, // Efficient broadcast part 1
		"e": PartE, // Efficient broadcast part 2
	}

	part := flag.String("part", "e", "Specifies the part of the challenge to run. For example its value could be a, b, c, d, e")
	flag.Parse()

	fn := solutions[*part]
	if fn == nil {
		fmt.Println("Invalid part specified:", *part)
		fmt.Println("It could be that the solution is not yet ready for that part")
		return
	}

	fn()
}

type BroadcastMessageBody struct {
	maelstrom.MessageBody
	Message  int   `json:"message"`
	Messages []int `json:"messages,omitempty"`
}

type ReadMessageBody struct {
	maelstrom.MessageBody
	Messages []int `json:"messages"`
}

type TopologyMessageBody struct {
	maelstrom.MessageBody
	Topology map[string][]string `json:"topology"`
}
