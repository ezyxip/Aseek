package main

import (
	"encoding/json"
	"fmt"

	"aseek-orchestrator/internal/ipc"
)

type stageInfo struct {
	Stage  string `json:"stage"`
	Detail string `json:"detail"`
	Count  int    `json:"count"`
}

var stages = []struct {
	key  string
	icon string
	label string
}{
	{"searching", "🔍", "Поиск"},
	{"reranking", "⚖", "Ранжирование"},
	{"prefill", "📦", "Префилл"},
	{"streaming", "✨", "Генерация"},
}

var pipelineDone bool

func printPipeline(msg *ipc.Message, done bool) {
	var si stageInfo
	json.Unmarshal(msg.Payload, &si)

	current := -1
	for i, s := range stages {
		if s.key == si.Stage {
			current = i
			break
		}
	}
	if current < 0 {
		return
	}

	detail := si.Detail
	if si.Count > 0 {
		detail = fmt.Sprintf("%s (%d)", si.Detail, si.Count)
	}

	fmt.Println()
	for i, s := range stages {
		switch {
		case done || i < current:
			fmt.Printf("  %s \033[32m✓ %s\033[0m\n", s.icon, s.label)
		case i == current:
			spin := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			_ = spin
			line := fmt.Sprintf("  %s \033[33m⟳ %s\033[0m", s.icon, s.label)
			if detail != "" {
				line += " \033[90m" + detail + "\033[0m"
			}
			fmt.Println(line)
		default:
			fmt.Printf("  %s \033[90m· %s\033[0m\n", s.icon, s.label)
		}
	}
	// rewind
	fmt.Printf("\033[%dA", len(stages)+1)
}

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
	case ipc.TypeStage:
		printPipeline(msg, false)
	case ipc.TypeToken:
		if !pipelineDone {
			fmt.Printf("\033[%dB", len(stages)+1)
			pipelineDone = true
		}
		fmt.Print(string(msg.Payload))
	case ipc.TypeSources:
		if !pipelineDone {
			fmt.Printf("\033[%dB", len(stages)+1)
			pipelineDone = true
		}
		printSources(msg.Payload)
	case ipc.TypeDone:
		if !pipelineDone {
			fmt.Printf("\033[%dB", len(stages)+1)
		}
		for _, s := range stages {
			fmt.Printf("  %s \033[32m✓ %s\033[0m\n", s.icon, s.label)
		}
		fmt.Println("\n[done]")
		pipelineDone = false
		*state = StateIdle
	case ipc.TypeError:
		if !pipelineDone {
			fmt.Printf("\033[%dB", len(stages)+1)
		} else {
			fmt.Println()
		}
		fmt.Printf("[error] %s\n", string(msg.Payload))
		pipelineDone = false
		*state = StateIdle
	case ipc.TypeBusy:
		if !pipelineDone {
			fmt.Printf("\033[%dB", len(stages)+1)
		}
		fmt.Println("\n[busy] Another request is already in progress")
		pipelineDone = false
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