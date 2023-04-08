package main

import (
	"fmt"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()

	id := 1
	n.Handle("generate", func(msg maelstrom.Message) error {
		uniqueId := fmt.Sprintf("%s-%d", n.ID(), id)
		id++

		return n.Reply(msg, struct {
			Type string `json:"type"`
			Id   string `json:"id"`
		}{Type: "generate_ok", Id: uniqueId})
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
