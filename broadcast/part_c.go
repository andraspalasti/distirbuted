package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func PartC() {
	n := maelstrom.NewNode()

	// We store the recieved messages here
	messages := map[int]struct{}{}
	unverified := map[string][]int{}
	mu := &sync.Mutex{}

	// Ticker to trigger a distribution of unverified messages
	ticker := time.NewTicker(time.Millisecond * 100)
	shouldDistribute := make(chan struct{})

	broadcast := func(dest string, message int) error {
		return n.RPC(dest,
			BroadcastMessageBody{
				MessageBody: maelstrom.MessageBody{Type: "broadcast"},
				Message:     message,
			},
			func(msg maelstrom.Message) error {
				mu.Lock()
				defer mu.Unlock()
				filtered := unverified[dest][:0]
				for _, m := range unverified[dest] {
					if m != message {
						filtered = append(filtered, m)
					}
				}
				unverified[dest] = filtered
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
		for _, nodeId := range n.NodeIDs() {
			if nodeId == msg.Src {
				return nil
			}
		}

		// Add to unverified list
		for _, nodeId := range n.NodeIDs() {
			if nodeId == n.ID() {
				continue
			}
			unverified[nodeId] = append(unverified[nodeId], body.Message)
		}

		// Trigger a distribution
		shouldDistribute <- struct{}{}
		ticker.Reset(time.Millisecond * 100)

		return nil
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		mu.Lock()
		defer mu.Unlock()

		buf := make([]int, 0, len(messages))
		for m, _ := range messages {
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
			select {
			case <-ticker.C:
			case <-shouldDistribute:
			}

			mu.Lock()
			for nodeId, messages := range unverified {
				for _, message := range messages {
					for _, neighbour := range n.NodeIDs() {
						if neighbour == nodeId {
							continue
						}
						broadcast(neighbour, message)
					}
				}
			}
			mu.Unlock()
		}
	}()

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
