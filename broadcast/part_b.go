package main

import (
	"encoding/json"
	"log"
	"slices"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func PartB() {
	n := maelstrom.NewNode()

	// We store the recieved messages here
	messages := []int{}
	mu := &sync.Mutex{}

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()
		messages = append(messages, body.Message)

		// Don't need to reply and rebroadcast if sender is a node
		if slices.Contains(n.NodeIDs(), msg.Src) {
			return nil
		}

		// Broadcast to all nodes except the sender
		for _, nodeId := range n.NodeIDs() {
			if nodeId == msg.Src || nodeId == n.ID() {
				continue
			}

			err := n.Send(nodeId, BroadcastMessageBody{
				MessageBody: maelstrom.MessageBody{Type: "broadcast"},
				Message:     body.Message,
			})
			if err != nil {
				return err
			}
		}
		return n.Reply(msg, maelstrom.MessageBody{Type: "broadcast_ok"})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		mu.Lock()
		defer mu.Unlock()

		return n.Reply(msg, ReadMessageBody{
			MessageBody: maelstrom.MessageBody{Type: "read_ok"},
			Messages:    messages,
		})
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		return n.Reply(msg, maelstrom.MessageBody{Type: "topology_ok"})
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
