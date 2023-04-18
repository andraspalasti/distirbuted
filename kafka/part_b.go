package main

import (
	"context"
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func readAndUpdate[T any](kv *maelstrom.KV, key string, fn func(T) T) error {
	res, err := kv.Read(context.Background(), key)
	if err != nil && maelstrom.ErrorCode(err) != maelstrom.KeyDoesNotExist {
		return err
	}
	log.Printf("current value %+v %T", res, res)

	data, _ := res.(T)
	err = kv.CompareAndSwap(context.Background(), key, data, fn(data), true)
	if err != nil {
		return err
	}
	return nil
}

func PartB() {
	node := maelstrom.NewNode()
	kv := maelstrom.NewLinKV(node)

	node.Handle("send", func(msg maelstrom.Message) error {
		var body struct {
			Type string `json:"type"`
			Key  string `json:"key"`
			Msg  int    `json:"msg"`
		}
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		offset := -1
		for {
			// Retry until successfull insertion
			err := readAndUpdate(kv, body.Key, func(msgs []any) []any {
				offset = len(msgs)
				return append(msgs, body.Msg)
			})
			if err == nil {
				break
			}
		}
		return node.Reply(msg, map[string]any{"type": "send_ok", "offset": offset})
	})

	node.Handle("poll", func(msg maelstrom.Message) error {
		var body struct {
			Type    string         `json:"type"`
			Offsets map[string]int `json:"offsets"`
		}
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		res := make(map[string][][2]int, len(body.Offsets))
		for key, offset := range body.Offsets {
			data, err := kv.Read(context.Background(), key)
			if err != nil {
				return err
			}

			msgs := data.([]any)

			numMsgs := len(msgs) - offset
			if 5 < numMsgs {
				numMsgs = 5
			}

			for i := offset; i < offset+numMsgs; i++ {
				res[key] = append(res[key], [2]int{i, int(msgs[i].(float64))})
			}
		}
		return node.Reply(msg, map[string]any{"type": "poll_ok", "msgs": res})
	})

	node.Handle("commit_offsets", func(msg maelstrom.Message) error {
		var body struct {
			Type    string         `json:"type"`
			Offsets map[string]int `json:"offsets"`
		}
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		commitOffset := func(key string, offset int) {
			for {
				err := readAndUpdate(kv, key+"_committed", func(oldOffset int) int {
					if offset < oldOffset {
						return oldOffset
					}
					return offset
				})
				if err == nil {
					break
				}
			}
		}

		for key, offset := range body.Offsets {
			go commitOffset(key, offset)
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

		offsets := make(map[string]int, len(body.Keys))
		for _, key := range body.Keys {
			offset, err := kv.Read(context.Background(), key+"_committed")
			if err != nil {
				return err
			}

			offsets[key] = offset.(int)
		}

		return node.Reply(msg, map[string]any{"type": "list_committed_offsets_ok", "offsets": offsets})
	})

	if err := node.Run(); err != nil {
		log.Fatal(err)
	}
}
