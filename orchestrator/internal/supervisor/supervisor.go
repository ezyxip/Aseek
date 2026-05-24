package supervisor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"aseek-orchestrator/internal/config"
	"aseek-orchestrator/internal/logging"
)

type State int

const (
	StateStopped State = iota
	StateStarting
	StateReady
	StateFailed
	StateRestarting
	StateFatal
)

func (s State) String() string {
	switch s {
	case StateStopped:
		return "STOPPED"
	case StateStarting:
		return "STARTING"
	case StateReady:
		return "READY"
	case StateFailed:
		return "FAILED"
	case StateRestarting:
		return "RESTARTING"
	case StateFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Supervisor struct {
	cfg       config.LlamaConfig
	cmd       *exec.Cmd
	mu        sync.Mutex
	state     State
	stateCh   chan State
	logWriter io.Writer
	cancel    context.CancelFunc
	log       *logging.Logger
}

func New(cfg config.LlamaConfig, logWriter io.Writer, log *logging.Logger) *Supervisor {
	return &Supervisor{
		cfg:       cfg,
		state:     StateStopped,
		stateCh:   make(chan State, 10),
		logWriter: logWriter,
		log:       log.WithModule("supervisor"),
	}
}

func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.state != StateStopped {
		s.mu.Unlock()
		return fmt.Errorf("supervisor not in stopped state: %s", s.state)
	}
	s.setStateLocked(StateStarting)
	s.mu.Unlock()

	llamaCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()

	args := s.buildArgs()

	cmd := exec.CommandContext(llamaCtx, s.cfg.Binary, args...)
	cmd.Stdout = s.logWriter
	cmd.Stderr = s.logWriter

	if err := cmd.Start(); err != nil {
		s.setState(StateFailed)
		return fmt.Errorf("start llama-server: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	go s.monitor(llamaCtx, cmd)
	return nil
}

func (s *Supervisor) WaitReady(ctx context.Context) error {
	if err := s.waitReady(ctx); err != nil {
		s.setState(StateFailed)
		return err
	}
	s.setState(StateReady)
	return nil
}

func (s *Supervisor) Stop() {
	s.mu.Lock()
	if s.state == StateStopped {
		s.mu.Unlock()
		return
	}
	s.setStateLocked(StateStopped)

	if s.cancel != nil {
		s.cancel()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	s.mu.Unlock()
}

func (s *Supervisor) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Supervisor) StateChanges() <-chan State {
	return s.stateCh
}

func (s *Supervisor) buildArgs() []string {
	args := []string{
		"--model", s.cfg.Model,
		"--port", fmt.Sprintf("%d", s.cfg.Port),
		"--ctx-size", fmt.Sprintf("%d", s.cfg.CtxSize),
		"--threads", fmt.Sprintf("%d", s.cfg.Threads),
		// "--slots", fmt.Sprintf("%d", s.cfg.Slots),
		"--parallel", fmt.Sprintf("%d", s.cfg.Slots),
		"--batch-size", fmt.Sprintf("%d", s.cfg.Batch),
	}
	if s.cfg.GPULayers > 0 {
		args = append(args, "--gpu-layers", fmt.Sprintf("%d", s.cfg.GPULayers))
	}
	return args
}

func (s *Supervisor) monitor(ctx context.Context, cmd *exec.Cmd) {
	cmd.Wait()

	s.mu.Lock()
	state := s.state
	s.mu.Unlock()

	if state == StateStopped {
		return
	}

	s.log.Warn("llama-server exited", "state", state)

	if err := s.restart(ctx); err != nil {
		s.log.Error("llama-server restart failed", "error", err)
		s.setState(StateFatal)
	}
}

func (s *Supervisor) restart(ctx context.Context) error {
	backoff := []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second}

	for attempt := 0; attempt < len(backoff); attempt++ {
		s.setState(StateRestarting)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff[attempt]):
		}

		s.log.Info("restarting llama-server", "attempt", attempt+1)
		s.setState(StateStarting)

		cmd := exec.CommandContext(ctx, s.cfg.Binary, s.buildArgs()...)
		cmd.Stdout = s.logWriter
		cmd.Stderr = s.logWriter

		if err := cmd.Start(); err != nil {
			s.log.Warn("restart attempt failed", "attempt", attempt+1, "error", err)
			continue
		}

		s.mu.Lock()
		s.cmd = cmd
		s.mu.Unlock()

		if err := s.waitReady(ctx); err != nil {
			s.log.Warn("restart health check failed", "attempt", attempt+1, "error", err)
			continue
		}

		s.setState(StateReady)
		s.log.Info("llama-server restarted successfully")

		go s.monitor(ctx, cmd)
		return nil
	}

	return fmt.Errorf("llama-server failed after %d restart attempts", len(backoff))
}

func (s *Supervisor) waitReady(ctx context.Context) error {
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", s.cfg.Port)
	client := &http.Client{Timeout: 5 * time.Second}

	backoff := []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second, 20 * time.Second}

	for attempt := 0; attempt < len(backoff); attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := client.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if err == nil {
			resp.Body.Close()
		}

		time.Sleep(backoff[attempt])
	}

	return fmt.Errorf("llama-server not ready after %d attempts", len(backoff))
}

func (s *Supervisor) setState(st State) {
	s.mu.Lock()
	s.setStateLocked(st)
	s.mu.Unlock()
}

func (s *Supervisor) setStateLocked(st State) {
	s.state = st
	select {
	case s.stateCh <- st:
	default:
	}
}
