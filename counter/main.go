package main

import (
	"context"
	"encoding/json"
	"log"
	"sync/atomic"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type addMessageBody struct {
	maelstrom.MessageBody
	Delta int `json:"delta"`
}

type readOkMessageBody struct {
	maelstrom.MessageBody
	Value int `json:"value"`
}

func readWithTimeout(kv *maelstrom.KV) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(50)*time.Millisecond)
	defer cancel()

	value, err := kv.ReadInt(ctx, "counter")
	if err != nil {
		return 0, err
	}
	return value, nil
}

func casWithTimeout(kv *maelstrom.KV, old, new int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(50)*time.Millisecond)
	defer cancel()

	return kv.CompareAndSwap(ctx, "counter", old, new, true)
}

func main() {
	node := maelstrom.NewNode()
	kv := maelstrom.NewSeqKV(node)

	lastRead := atomic.Int32{}
	delta := atomic.Int32{}

	node.Handle("add", func(msg maelstrom.Message) error {
		var body addMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return nil
		}

		delta.Add(int32(body.Delta))
		return node.Reply(msg, maelstrom.MessageBody{Type: "add_ok"})
	})

	node.Handle("read", func(msg maelstrom.Message) error {
		return node.Reply(msg, readOkMessageBody{
			MessageBody: maelstrom.MessageBody{Type: "read_ok"},
			Value:       int(lastRead.Load()),
		})
	})

	// Periodically read the counter's value to update our value
	// and send our updates to the KV service
	go func() {
		for {
			<-time.After(time.Duration(100) * time.Millisecond)

			value, err := readWithTimeout(kv)
			if err != nil && maelstrom.ErrorCode(err) != maelstrom.KeyDoesNotExist {
				continue
			}

			// Store the read value
			lastRead.Store(int32(value))

			d := delta.Load()
			if d == 0 {
				// There is no point of updating if it won't come with a change
				continue
			}

			err = casWithTimeout(kv, value, value+int(d))
			if err != nil && maelstrom.ErrorCode(err) != maelstrom.KeyDoesNotExist {
				continue
			}

			// We succcessfully changed the counter's value in the KV service
			// so we should update our side too
			lastRead.Store(int32(value) + d)
			delta.Add(-d)
		}
	}()

	if err := node.Run(); err != nil {
		log.Fatal(err)
	}
}
