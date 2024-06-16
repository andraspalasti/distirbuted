package main

import (
	"encoding/json"
	"log"
	"slices"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func PartE() {
	n := maelstrom.NewNode()

	// We store the recieved messages here
	messages := map[int]struct{}{}
	unverified := map[string][]int{}
	mu := &sync.Mutex{}

	// Ticker to trigger a distribution of unverified messages
	ticker := time.NewTicker(time.Millisecond * 800)

	broadcast := func(nodeId string, messages []int) error {
		return n.RPC(nodeId,
			BroadcastMessageBody{
				MessageBody: maelstrom.MessageBody{Type: "broadcast"},
				Messages:    messages, Message: messages[0],
			},
			func(msg maelstrom.Message) error {
				mu.Lock()
				defer mu.Unlock()
				if slices.Equal(messages, unverified[nodeId][:len(messages)]) {
					unverified[nodeId] = unverified[nodeId][len(messages):]
				}
				return nil
			},
		)
	}

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		body.Messages = append(body.Messages, body.Message)

		// Confirm to the sender that we recieved the message
		err := n.Reply(msg, maelstrom.MessageBody{Type: "broadcast_ok"})
		if err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()

		// Store the messages
		for _, m := range body.Messages {
			messages[m] = struct{}{}
		}

		// Don't need to rebroadcast if sender is a node
		if slices.Contains(n.NodeIDs(), msg.Src) {
			return nil
		}

		// Add to unverified list
		for _, nodeId := range n.NodeIDs() {
			if nodeId == n.ID() {
				continue
			}
			unverified[nodeId] = append(unverified[nodeId], body.Messages...)
		}
		return nil
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		mu.Lock()
		defer mu.Unlock()

		buf := make([]int, 0, len(messages))
		for m := range messages {
			buf = append(buf, m)
		}

		return n.Reply(msg, ReadMessageBody{
			MessageBody: maelstrom.MessageBody{Type: "read_ok"},
			Messages:    buf,
		})
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		return n.Reply(msg, maelstrom.MessageBody{Type: "topology_ok"})
	})

	go func() {
		for {
			<-ticker.C
			mu.Lock()
			for nodeId, messages := range unverified {
				if len(messages) == 0 {
					continue
				}
				broadcast(nodeId, slices.Clone(messages))
			}
			mu.Unlock()
		}
	}()

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
