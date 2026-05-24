package main

import (
	"fmt"

	"aseek-orchestrator/internal/ipc"
)

type LoopState int

const (
	StateIdle LoopState = iota
	StateStreaming
	StateCancelling
)

func handleMessage(msg *ipc.Message) {
	switch msg.Header.Type {
	case ipc.TypeBusy:
		fmt.Println("[busy] Another request is already in progress")
	case ipc.TypePong:
		fmt.Println("[pong]")
	case ipc.TypeProfileList:
		fmt.Printf("[profiles] %s\n", string(msg.Payload))
	case ipc.TypeError:
		fmt.Printf("[error] %s\n", string(msg.Payload))
	}
}

func handleStreamMessage(msg *ipc.Message, state *LoopState) {
	switch msg.Header.Type {
	case ipc.TypeToken:
		fmt.Print(string(msg.Payload))
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