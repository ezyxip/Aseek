package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"aseek-orchestrator/internal/config"
	"aseek-orchestrator/internal/ipc"
	"aseek-orchestrator/internal/llama"
	"aseek-orchestrator/internal/logging"
	"aseek-orchestrator/internal/pipeline"
	"aseek-orchestrator/internal/profile"
	"aseek-orchestrator/internal/prompt"
	"aseek-orchestrator/internal/request"
	"aseek-orchestrator/internal/streaming"
	"aseek-orchestrator/internal/supervisor"
)

func main() {
	cfgPath := config.DefaultPath()
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfgManager := config.NewManager(cfgPath)
	cfg, err := cfgManager.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() {
		closeOnce.Do(func() {
			close(done)
		})
	}

	log := logging.New(os.Stderr, logging.ParseLevel(cfg.Logging.Level), false)

	profMgr := profile.New(config.DefaultProfilesPath(), log)
	if err := profMgr.Load(); err != nil {
		log.Warn("profiles not loaded", "error", err)
	}

	promptBuilder := prompt.New(config.DefaultTemplatesDir(), log)
	if err := promptBuilder.LoadTemplates(); err != nil {
		log.Warn("templates not loaded", "error", err)
	}

	streamEng := streaming.New(cfg.FlushInterval(), log)
	pipeln := pipeline.New(cfg.RequestTimeout(), profMgr, log, cfg.Reranker.URL)
	sup := supervisor.New(cfg.Llama, os.Stderr, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sup.Start(ctx); err != nil {
		log.Error("supervisor start failed", "error", err)
		os.Exit(1)
	}

	if err := sup.WaitReady(ctx); err != nil {
		log.Error("llama-server not ready", "error", err)
		sup.Stop()
		os.Exit(1)
	}

	log.Info("llama-server ready", "port", cfg.Llama.Port)

	reqMgr := request.New(ctx, sup, pipeln, promptBuilder, streamEng, llama.New(
		fmt.Sprintf("http://127.0.0.1:%d", cfg.Llama.Port),
		&http.Client{},
		log,
	), profMgr, log)

	sockDir, err := socketDir()
	if err != nil {
		log.Error("socket dir", "error", err)
		sup.Stop()
		os.Exit(1)
	}

	ipcSrv, err := ipc.NewServer(sockDir, reqMgr.Handle)
	if err != nil {
		log.Error("ipc server failed", "error", err)
		sup.Stop()
		os.Exit(1)
	}

	reqMgr.SetIPCServer(ipcSrv)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			log.Info("shutting down")
		case <-done:
			return
		}
		cancel()
		ipcSrv.Close()
		sup.Stop()
		closeDone()
	}()

	log.Info("waiting for client", "socket", ipcSrv.Path())

	go func() {
		if err := ipcSrv.Accept(); err != nil {
			log.Warn("accept ended", "error", err)
			closeDone()
			return
		}
		log.Info("client connected")

		for {
			if err := ipcSrv.Serve(); err != nil {
				log.Warn("serve ended", "error", err)
				return
			}
		}
	}()

	<-done
	sup.Stop()
}

func socketDir() (string, error) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		return "", fmt.Errorf("XDG_RUNTIME_DIR not set")
	}
	return dir, nil
}