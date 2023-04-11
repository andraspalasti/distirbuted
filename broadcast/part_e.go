package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type MultiBroadcastMessageBody struct {
	BroadcastMessageBody
	Messages []int `json:"messages"`
}

func (s *Server) MultiBroadcastLoop(sendTo []string, messages []int) {
	respCh := make(chan struct {
		dest   string
		failed bool
	})

	for 0 < len(sendTo) {
		for _, neighbour := range sendTo {
			go func(dest string) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(250)*time.Millisecond)
				defer cancel()

				_, err := s.SyncRPC(ctx, dest, MultiBroadcastMessageBody{
					BroadcastMessageBody: BroadcastMessageBody{
						MessageBody: maelstrom.MessageBody{Type: "broadcast"},
					},
					Messages: messages,
				})
				respCh <- struct {
					dest   string
					failed bool
				}{dest: dest, failed: err != nil}
			}(neighbour)
		}

		failed := make([]string, 0, len(sendTo))
		for i := 0; i < len(sendTo); i++ {
			broadcast := <-respCh
			if broadcast.failed {
				failed = append(failed, broadcast.dest)
			}
		}
		sendTo = failed
	}
}

func PartE() {
	s := NewServer()

	s.Handle("broadcast", func(msg maelstrom.Message) error {
		var body MultiBroadcastMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		err := s.Reply(msg, maelstrom.MessageBody{Type: "broadcast_ok"})
		if err != nil {
			return err
		}

		if len(body.Messages) == 0 {
			body.Messages = append(body.Messages, body.Message)
		}

		s.mu.Lock()
		for _, message := range body.Messages {
			if _, ok := s.messages[message]; !ok {
				s.messages[message] = false
			}
		}
		s.mu.Unlock()

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

	// Periodically gossip unbroadcasted messages
	go func() {
		ticker := time.NewTicker(time.Duration(500) * time.Millisecond)
		for {
			unsent := make([]int, 0)
			s.mu.Lock()
			for message, sent := range s.messages {
				if !sent {
					unsent = append(unsent, message)
					s.messages[message] = true // Don't try to broadcast it again
				}
			}
			s.mu.Unlock()

			go s.MultiBroadcastLoop(s.neighbours, unsent)

			<-ticker.C
		}
	}()

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
