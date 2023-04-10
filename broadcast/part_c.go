package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

// Broadcast a message to all neighbouring nodes until all of them respond with a "broadcast_ok"
func (s *Server) BroadcastLoop(src string, message int) {
	sendTo := s.neighbours
	respCh := make(chan struct {
		dest   string
		failed bool
	})

	for {
		for _, neighbour := range sendTo {
			if neighbour == src {
				continue
			}

			go func(dest string) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(50)*time.Millisecond)
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

		fails := make([]string, 0, len(sendTo))
		for i := 0; i < len(sendTo); i++ {
			resp := <-respCh
			if resp.failed {
				fails = append(fails, resp.dest)
			}
		}

		if len(fails) == 0 {
			break
		}
		sendTo = fails
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

		s.Save(body.Message)
		s.BroadcastLoop(msg.Src, body.Message)
		return nil
	})
	s.Handle("topology", s.TopologyHandler)
	s.Handle("read", s.ReadHandler)

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
