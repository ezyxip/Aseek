package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"aseek-orchestrator/internal/ipc"
)

func main() {
	socketPath := ""
	if len(os.Args) > 1 {
		socketPath = os.Args[1]
	}

	client, err := Dial(socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connection failed: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	inputCh := make(chan string)
	scanner := bufio.NewScanner(os.Stdin)
	go func() {
		for scanner.Scan() {
			inputCh <- scanner.Text()
		}
		close(inputCh)
	}()

	var reqID uint32
	state := StateIdle

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)

	var cancelTimer <-chan time.Time

	prompt := func() {
		fmt.Print("\n[aseek]> ")
	}

	prompt()

	for {
		select {
		case msg := <-client.Recv():
			if msg == nil {
				fmt.Println("\n[disconnected]")
				os.Exit(1)
			}

			if state == StateStreaming || state == StateCancelling {
				handleStreamMessage(msg, &state)
			} else {
				handleMessage(msg)
			}

			if state == StateIdle {
				cancelTimer = nil
				prompt()
			}

		case <-sigCh:
			if state == StateStreaming {
				client.Send(ipc.NewMessage(ipc.TypeCancel, reqID, nil))
				state = StateCancelling
				cancelTimer = time.After(3 * time.Second)
				fmt.Print("\n[cancelling...]")
			}

		case <-cancelTimer:
			if state == StateCancelling {
				fmt.Print("\n[cancel timeout]")
			}
			state = StateIdle
			cancelTimer = nil
			prompt()

		case line, ok := <-inputCh:
			if !ok {
				if state == StateStreaming || state == StateCancelling {
					client.Send(ipc.NewMessage(ipc.TypeCancel, reqID, nil))
					time.Sleep(3 * time.Second)
				}
				fmt.Println()
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				if state == StateIdle {
					prompt()
				}
				continue
			}

			if state == StateStreaming || state == StateCancelling {
				fmt.Println("[busy] finish or cancel the current request first")
				continue
			}

			parts := strings.Fields(line)
			cmd := parts[0]

			switch cmd {
			case "exit", "quit":
				fmt.Println()
				return

			case "help":
				fmt.Println(`Commands:
  query <text>   Send a RAG query
  cancel         Cancel current request
  ping           Check connection
  profiles       List profiles
  switch <name>  Switch profile
  help           Show this help
  exit / quit    Exit`)

			case "query":
				payload := strings.Join(parts[1:], " ")
				if payload == "" {
					fmt.Println("[error] query text required")
					prompt()
					continue
				}
				reqID++
				state = StateStreaming
				err := client.Send(ipc.NewMessage(ipc.TypeQuery, reqID, []byte(payload)))
				if err != nil {
					fmt.Printf("[error] send: %v\n", err)
					state = StateIdle
				}

			case "cancel":
				fmt.Println("[idle] no active request")

			case "ping":
				client.Send(ipc.NewMessage(ipc.TypePing, 0, nil))

			case "profiles":
				client.Send(ipc.NewMessage(ipc.TypeProfileList, 0, nil))

			case "switch":
				if len(parts) < 2 {
					fmt.Println("[error] profile name required")
				} else {
					payload := fmt.Sprintf(`{"name":%q}`, parts[1])
					client.Send(ipc.NewMessage(ipc.TypeProfileSwitch, 0, []byte(payload)))
				}

			default:
				fmt.Printf("[error] unknown command: %s\n", cmd)
			}

			if state == StateIdle {
				prompt()
			}
		}
	}
}