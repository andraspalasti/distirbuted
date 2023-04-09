package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

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

type WriteMessageBody struct {
	maelstrom.MessageBody
	Messages []int `json:"messages"`
}

type MsgStore[T any] struct {
	sync.Mutex
	messages []T
}

func NewMsgStore[T any]() *MsgStore[T] {
	return &MsgStore[T]{
		Mutex:    sync.Mutex{},
		messages: []T{},
	}
}

func (ms *MsgStore[T]) Read() []T {
	ms.Lock()
	defer ms.Unlock()

	messages := make([]T, len(ms.messages))
	copy(messages, ms.messages)
	return messages
}

func (ms *MsgStore[T]) Put(messages ...T) {
	ms.Lock()
	defer ms.Unlock()

	ms.messages = append(ms.messages, messages...)
}

func main() {
	n := maelstrom.NewNode()

	// We store all the messages here
	messages := NewMsgStore[int]()

	unbroadcasted := NewMsgStore[int]()

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		unbroadcasted.Put(body.Message)

		return n.Reply(msg, maelstrom.MessageBody{Type: "broadcast_ok"})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		return n.Reply(msg, ReadMessageBody{
			MessageBody: maelstrom.MessageBody{Type: "read_ok"},
			Messages:    messages.Read(),
		})
	})

	n.Handle("write", func(msg maelstrom.Message) error {
		var body WriteMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		messages.Put(body.Messages...)
		return nil
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		return n.Reply(msg, maelstrom.MessageBody{Type: "topology_ok"})
	})

	// From time to time broadcast the unbroadcasted messages to each node
	go func() {
		ticker := time.NewTicker(time.Duration(200) * time.Millisecond)
		for {
			shouldWrite := unbroadcasted.Read()

			for _, nodeId := range n.NodeIDs() {
				if n.ID() == nodeId {
					continue
				}

				n.Send(nodeId, WriteMessageBody{
					MessageBody: maelstrom.MessageBody{Type: "write"},
					Messages:    shouldWrite,
				})
			}

			messages.Put(shouldWrite...)
			<-ticker.C
		}
	}()

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
