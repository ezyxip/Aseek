package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"aseek-orchestrator/internal/ipc"
)

type Client struct {
	conn   net.Conn
	recv   chan *ipc.Message
	closed bool
}

func Dial(path string) (*Client, error) {
	if path == "" {
		rd := os.Getenv("XDG_RUNTIME_DIR")
		if rd == "" {
			return nil, fmt.Errorf("XDG_RUNTIME_DIR not set and no socket path provided")
		}
		path = filepath.Join(rd, "aurora-rag.sock")
	}

	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", path, err)
	}

	c := &Client{
		conn: conn,
		recv: make(chan *ipc.Message, 64),
	}

	go c.readLoop()

	return c, nil
}

func (c *Client) readLoop() {
	defer close(c.recv)
	for {
		msg, err := ipc.ReadMessage(c.conn)
		if err != nil {
			if !c.closed {
				c.recv <- nil
			}
			return
		}
		c.recv <- msg
	}
}

func (c *Client) Send(msg *ipc.Message) error {
	_, err := c.conn.Write(msg.Encode())
	return err
}

func (c *Client) Recv() <-chan *ipc.Message {
	return c.recv
}

func (c *Client) Close() {
	c.closed = true
	c.conn.Close()
}