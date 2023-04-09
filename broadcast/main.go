package main

import (
	"flag"
	"fmt"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type BroadcastMessageBody struct {
	maelstrom.MessageBody
	Message int `json:"message"`
}

type ReadMessageBody struct {
	maelstrom.MessageBody
	Messages []int `json:"messages"`
}

type TopologyMessageBody struct {
	maelstrom.MessageBody
	Topology map[string][]string `json:"topology"`
}

func main() {
	solutions := map[string]func(){"a": PartA, "b": PartB}

	part := flag.String("part", "b", "Specifies the part of the challenge to run. For example its value could be a, b, c, d, e")
	flag.Parse()

	fn := solutions[*part]
	if fn == nil {
		fmt.Println("Invalid part specified:", *part)
		fmt.Println("It could be that the solution is not yet ready for that part")
		return
	}

	fn()
}
