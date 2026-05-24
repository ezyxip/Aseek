package streaming

import (
	"context"
	"time"

	"aseek-orchestrator/internal/logging"
)

type Engine struct {
	flushInterval time.Duration
	log           *logging.Logger
}

func New(flushInterval time.Duration, log *logging.Logger) *Engine {
	return &Engine{
		flushInterval: flushInterval,
		log:           log.WithModule("streaming"),
	}
}

func (e *Engine) Stream(ctx context.Context, tokens <-chan string, send func([]byte) error) error {
	ticker := time.NewTicker(e.flushInterval)
	defer ticker.Stop()

	var buf []byte

	flush := func() error {
		if len(buf) == 0 {
			return nil
		}
		if err := send(buf); err != nil {
			return err
		}
		buf = buf[:0]
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return flush()
		case token, ok := <-tokens:
			if !ok {
				return flush()
			}
			buf = append(buf, []byte(token)...)
		case <-ticker.C:
			if err := flush(); err != nil {
				return err
			}
		}
	}
}