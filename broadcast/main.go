package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sync"

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

type TopologyMessageBody struct {
	maelstrom.MessageBody
	Topology map[string][]string `json:"topology"`
}

type Server struct {
	*maelstrom.Node
	neighbours []string

	mu       sync.Mutex
	messages map[int]struct{}
}

func NewServer() *Server {
	return &Server{
		Node:     maelstrom.NewNode(),
		messages: make(map[int]struct{}),
	}
}

// Saves the specified message and returns true
// if we already had this message in the store else it returns false.
func (s *Server) Save(message int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.messages[message]
	s.messages[message] = struct{}{}
	return ok
}

func (s *Server) TopologyHandler(msg maelstrom.Message) error {
	var body TopologyMessageBody
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.neighbours = body.Topology[s.ID()]
	return s.Reply(msg, maelstrom.MessageBody{Type: "topology_ok"})
}

func (s *Server) ReadHandler(msg maelstrom.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	buf := make([]int, 0, len(s.messages))
	for message := range s.messages {
		buf = append(buf, message)
	}

	return s.Reply(msg, ReadMessageBody{
		MessageBody: maelstrom.MessageBody{Type: "read_ok"},
		Messages:    buf,
	})
}

func main() {
	solutions := map[string]func(){"a": PartA, "b": PartB, "c": PartC}

	part := flag.String("part", "c", "Specifies the part of the challenge to run. For example its value could be a, b, c, d, e")
	flag.Parse()

	fn := solutions[*part]
	if fn == nil {
		fmt.Println("Invalid part specified:", *part)
		fmt.Println("It could be that the solution is not yet ready for that part")
		return
	}

	fn()
}
