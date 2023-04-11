package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

// Broadcast a message to all neighbouring nodes until all of them respond with a "broadcast_ok"
func (s *Server) BroadcastLoop(sendTo []string, message int) {
	respCh := make(chan struct {
		dest   string
		failed bool
	})

	for 0 < len(sendTo) {
		for _, neighbour := range sendTo {
			go func(dest string) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(250)*time.Millisecond)
				defer cancel()

				_, err := s.SyncRPC(ctx, dest, BroadcastMessageBody{
					MessageBody: maelstrom.MessageBody{Type: "broadcast"},
					Message:     message,
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

func PartC() {
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
	s.Handle("topology", s.TopologyHandler)
	s.Handle("read", s.ReadHandler)

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
