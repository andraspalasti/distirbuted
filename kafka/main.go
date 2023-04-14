package main

import (
	"encoding/json"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	node := maelstrom.NewNode()

	mu := sync.Mutex{}
	messages := make(map[string][]int)
	committed := make(map[string]int)

	node.Handle("send", func(msg maelstrom.Message) error {
		var body struct {
			Type string `json:"type"`
			Key  string `json:"key"`
			Msg  int    `json:"msg"`
		}
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()

		messages[body.Key] = append(messages[body.Key], body.Msg)
		return node.Reply(msg, map[string]any{"type": "send_ok", "offset": len(messages[body.Key]) - 1})
	})

	node.Handle("poll", func(msg maelstrom.Message) error {
		var body struct {
			Type    string         `json:"type"`
			Offsets map[string]int `json:"offsets"`
		}
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()

		msgs := make(map[string][][2]int, len(body.Offsets))
		for key, offset := range body.Offsets {
			for i := offset; i < len(messages[key]); i++ {
				msgs[key] = append(msgs[key], [2]int{i, messages[key][i]})
			}
		}
		return node.Reply(msg, map[string]any{"type": "poll_ok", "msgs": msgs})
	})

	node.Handle("commit_offsets", func(msg maelstrom.Message) error {
		var body struct {
			Type    string         `json:"type"`
			Offsets map[string]int `json:"offsets"`
		}
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()

		for key, offset := range body.Offsets {
			committed[key] = offset
		}

		return node.Reply(msg, map[string]any{"type": "commit_offsets_ok"})
	})

	node.Handle("list_committed_offsets", func(msg maelstrom.Message) error {
		var body struct {
			Type string   `json:"type"`
			Keys []string `json:"keys"`
		}
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()

		offsets := make(map[string]int, len(body.Keys))
		for _, key := range body.Keys {
			offsets[key] = committed[key]
		}
		return node.Reply(msg, map[string]any{"type": "list_committed_offsets_ok", "offsets": offsets})
	})

	if err := node.Run(); err != nil {
		log.Fatal(err)
	}
}
