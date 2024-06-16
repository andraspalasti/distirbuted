package main

import (
	"encoding/json"
	"log"
	"slices"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func PartD() {
	n := maelstrom.NewNode()

	// We store the recieved messages here
	messages := map[int]struct{}{}
	unverified := map[string][]int{}
	mu := &sync.Mutex{}

	// Ticker to trigger a distribution of unverified messages
	ticker := time.NewTicker(time.Millisecond * 250)

	broadcast := func(dest string, message int) error {
		return n.RPC(dest,
			BroadcastMessageBody{
				MessageBody: maelstrom.MessageBody{Type: "broadcast"},
				Message:     message,
			},
			func(msg maelstrom.Message) error {
				mu.Lock()
				defer mu.Unlock()
				unverified[dest] = slices.DeleteFunc(
					unverified[dest],
					func(m int) bool { return m == message },
				)
				return nil
			},
		)
	}

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// Confirm to the sender that we recieved the message
		err := n.Reply(msg, maelstrom.MessageBody{Type: "broadcast_ok"})
		if err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()

		// Check if we already have the message
		if _, ok := messages[body.Message]; ok {
			return nil
		}

		// Store the message
		messages[body.Message] = struct{}{}

		// Don't need to rebroadcast if sender is a node
		if slices.Contains(n.NodeIDs(), msg.Src) {
			return nil
		}

		// Add to unverified list
		for _, nodeId := range n.NodeIDs() {
			if nodeId == n.ID() {
				continue
			}
			unverified[nodeId] = append(unverified[nodeId], body.Message)
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
				for _, message := range messages {
					broadcast(nodeId, message)
				}
			}
			mu.Unlock()
		}
	}()

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
