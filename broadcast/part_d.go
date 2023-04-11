package main

import (
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func PartD() {
	s := NewServer()

	s.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		err := s.Reply(msg, maelstrom.MessageBody{Type: "broadcast_ok"})
		if err != nil {
			return err
		}

		if !s.Save(body.Message) {
			sendTo := make([]string, 0, len(s.neighbours))
			for _, neighbour := range s.neighbours {
				if neighbour != msg.Src {
					sendTo = append(sendTo, neighbour)
				}
			}

			s.BroadcastLoop(sendTo, body.Message)
		}
		return nil
	})

	s.Handle("topology", func(msg maelstrom.Message) error {
		// Create our own topology to reduce latency
		// There will be one master node that is connected to all others and every message goes through it
		if s.ID() == s.NodeIDs()[0] {
			s.neighbours = append(s.neighbours, s.NodeIDs()[1:]...)
		} else {
			s.neighbours = append(s.neighbours, s.NodeIDs()[0])
		}

		return s.Reply(msg, maelstrom.MessageBody{Type: "topology_ok"})
	})

	s.Handle("read", s.ReadHandler)

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
