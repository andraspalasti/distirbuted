package main

import (
	"encoding/json"
	"log"
	"sync"

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
	n := maelstrom.NewNode()

	var topology map[string][]string

	// We store all the messages here
	messages := make(map[int]struct{})
	mu := &sync.Mutex{}

	// Saves the message and returns true if it was already in the map else false
	save := func(message int) bool {
		mu.Lock()
		defer mu.Unlock()

		if _, ok := messages[message]; ok {
			return true
		}
		messages[message] = struct{}{}
		return false
	}

	// Reads all the messages into a slice and returns it
	readAll := func() []int {
		mu.Lock()
		defer mu.Unlock()

		buf := make([]int, 0, len(messages))
		for message := range messages {
			buf = append(buf, message)
		}
		return buf
	}

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		if save(body.Message) {
			return nil
		}

		// Broadcast message for neighbours
		for _, nodeId := range topology[n.ID()] {
			if nodeId == msg.Src {
				continue
			}

			if err := n.Send(nodeId, BroadcastMessageBody{
				MessageBody: maelstrom.MessageBody{Type: "broadcast"},
				Message:     body.Message,
			}); err != nil {
				return err
			}
		}

		// If the src is a node than we dont have to send a broadcast_ok reply
		if _, ok := topology[msg.Src]; ok {
			return nil
		}
		return n.Reply(msg, maelstrom.MessageBody{Type: "broadcast_ok"})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		return n.Reply(msg, ReadMessageBody{
			MessageBody: maelstrom.MessageBody{Type: "read_ok"},
			Messages:    readAll(),
		})
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		var body TopologyMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		topology = body.Topology
		return n.Reply(msg, maelstrom.MessageBody{Type: "topology_ok"})
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
