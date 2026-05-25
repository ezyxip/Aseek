package main

import (
	"encoding/json"
	"fmt"

	"aseek-orchestrator/internal/ipc"
)

type LoopState int

const (
	StateIdle LoopState = iota
	StateStreaming
	StateCancelling
)

type sourceDoc struct {
	Index  int     `json:"index"`
	Title  string  `json:"title"`
	Source string  `json:"source"`
	Score  float64 `json:"score"`
}

func handleMessage(msg *ipc.Message) {
	switch msg.Header.Type {
	case ipc.TypeBusy:
		fmt.Println("[busy] Another request is already in progress")
	case ipc.TypePong:
		fmt.Println("[pong]")
	case ipc.TypeProfileList:
		fmt.Printf("[profiles] %s\n", string(msg.Payload))
	case ipc.TypeSources:
		printSources(msg.Payload)
	case ipc.TypeError:
		fmt.Printf("[error] %s\n", string(msg.Payload))
	}
}

func handleStreamMessage(msg *ipc.Message, state *LoopState) {
	switch msg.Header.Type {
	case ipc.TypeToken:
		fmt.Print(string(msg.Payload))
	case ipc.TypeSources:
		printSources(msg.Payload)
	case ipc.TypeDone:
		fmt.Println("\n[done]")
		*state = StateIdle
	case ipc.TypeError:
		fmt.Printf("\n[error] %s\n", string(msg.Payload))
		*state = StateIdle
	case ipc.TypeBusy:
		fmt.Println("\n[busy] Another request is already in progress")
		*state = StateIdle
	}
}

func printSources(data []byte) {
	var srcs []sourceDoc
	if err := json.Unmarshal(data, &srcs); err != nil {
		return
	}
	if len(srcs) == 0 {
		return
	}
	fmt.Println("\n[sources]")
	for _, s := range srcs {
		title := s.Title
		if title == "" {
			title = fmt.Sprintf("[%d]", s.Index)
		}
		if s.Source != "" {
			fmt.Printf("  %s — %s\n", title, s.Source)
		} else {
			fmt.Printf("  %s (no source)\n", title)
		}
	}
}