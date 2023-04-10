package main

import (
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func PartB() {
	s := NewServer()

	s.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		if s.Save(body.Message) {
			return nil
		}

		// Broadcast message for neighbours
		for _, nodeId := range s.neighbours {
			if nodeId == msg.Src {
				continue
			}

			s.Send(nodeId, BroadcastMessageBody{
				MessageBody: maelstrom.MessageBody{Type: "broadcast"},
				Message:     body.Message,
			})
		}

		// We don't use MsgIDs for inter communication
		// so we know that this is coming from a node
		if body.MsgID == 0 {
			return nil
		}
		return s.Reply(msg, maelstrom.MessageBody{Type: "broadcast_ok"})
	})
	s.Handle("read", s.ReadHandler)
	s.Handle("topology", s.TopologyHandler)

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
