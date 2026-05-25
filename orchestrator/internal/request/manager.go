package request

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"aseek-orchestrator/internal/ipc"
	"aseek-orchestrator/internal/llama"
	"aseek-orchestrator/internal/logging"
	"aseek-orchestrator/internal/pipeline"
	"aseek-orchestrator/internal/profile"
	"aseek-orchestrator/internal/prompt"
	"aseek-orchestrator/internal/streaming"
	"aseek-orchestrator/internal/supervisor"
)

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func jsonError(code, msg string) []byte {
	data, _ := json.Marshal(errorPayload{Code: code, Message: msg})
	return data
}

type Manager struct {
	mu           sync.Mutex
	active       bool
	currentReqID uint32
	currentGen   uint64
	cancel       context.CancelFunc

	baseCtx    context.Context
	supervisor *supervisor.Supervisor
	pipeline   *pipeline.Pipeline
	prompt     *prompt.Builder
	streaming  *streaming.Engine
	llama      *llama.Client
	profiles   *profile.Manager
	ipc        *ipc.Server
	log        *logging.Logger
}

func New(
	ctx context.Context,
	sup *supervisor.Supervisor,
	pl *pipeline.Pipeline,
	pb *prompt.Builder,
	se *streaming.Engine,
	lc *llama.Client,
	pm *profile.Manager,
	log *logging.Logger,
) *Manager {
	return &Manager{
		baseCtx:    ctx,
		supervisor: sup,
		pipeline:   pl,
		prompt:     pb,
		streaming:  se,
		llama:      lc,
		profiles:   pm,
		log:        log.WithModule("request"),
	}
}

func (m *Manager) SetIPCServer(srv *ipc.Server) {
	m.ipc = srv
}

func (m *Manager) Handle(msg *ipc.Message) {
	switch msg.Header.Type {
	case ipc.TypeQuery:
		m.handleQuery(msg)
	case ipc.TypeCancel:
		m.handleCancel(msg)
	case ipc.TypePing:
		m.ipc.Send(ipc.NewMessage(ipc.TypePong, msg.Header.RequestID, []byte(`{"status":"ok"}`)))
	case ipc.TypeProfileList:
		m.handleProfileList(msg)
	case ipc.TypeProfileSwitch:
		m.handleProfileSwitch(msg)
	default:
		m.ipc.SendError(msg.Header.RequestID, jsonError("UNKNOWN_TYPE",
			fmt.Sprintf("unknown message type: %d", msg.Header.Type)))
	}
}

func (m *Manager) handleQuery(msg *ipc.Message) {
	m.mu.Lock()
	if m.active {
		m.mu.Unlock()
		m.ipc.Send(ipc.NewMessage(ipc.TypeBusy, msg.Header.RequestID,
			[]byte(`{"code":"BUSY","message":"Another request is already in progress"}`)))
		return
	}

	if st := m.supervisor.State(); st != supervisor.StateReady {
		m.mu.Unlock()
		m.ipc.SendError(msg.Header.RequestID,
			jsonError("LLAMA_NOT_READY", fmt.Sprintf("llama-server is %s", st)))
		return
	}

	m.active = true
	reqID := msg.Header.RequestID
	m.currentReqID = reqID
	genID := m.currentGen + 1
	m.currentGen = genID
	m.mu.Unlock()

	ctx, cancel := context.WithCancel(m.baseCtx)
	m.mu.Lock()
	m.cancel = cancel
	m.mu.Unlock()

	query := string(msg.Payload)

	go m.runPipeline(ctx, query, reqID, genID)
}

func (m *Manager) handleCancel(msg *ipc.Message) {
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()

	m.ipc.Send(ipc.NewMessage(ipc.TypeDone, msg.Header.RequestID, nil))
}

func (m *Manager) runPipeline(ctx context.Context, query string, reqID uint32, genID uint64) {
	defer func() {
		m.mu.Lock()
		if m.currentGen == genID {
			m.active = false
			m.cancel = nil
		}
		m.mu.Unlock()
	}()

	m.log.Info("starting RAG pipeline")

	systemPrompt, err := m.prompt.BuildSystemPrompt(ctx)
	if err != nil {
		m.sendError(fmt.Sprintf("build system prompt: %v", err), reqID)
		return
	}

	if profilePrompt := m.profiles.ActivePrompt(); profilePrompt != "" {
		systemPrompt += "\n\n" + profilePrompt
	}

	docs, err := m.pipeline.ExecuteWithCallback(ctx, query, func(stage, detail string, count int) {
		payload, _ := json.Marshal(map[string]interface{}{
			"stage": stage,
			"detail": detail,
			"count":  count,
		})
		if err := m.ipc.Send(ipc.NewMessage(ipc.TypeStage, reqID, payload)); err != nil {
			m.log.Warn("send stage", "error", err)
		}
	})
	if err != nil {
		m.sendError(fmt.Sprintf("pipeline: %v", err), reqID)
		return
	}

	ragPrompt, err := m.prompt.BuildRAGPrompt(ctx, query, docs)
	if err != nil {
		m.sendError(fmt.Sprintf("build rag prompt: %v", err), reqID)
		return
	}

	m.sendStage(reqID, "prefill", "Подготовка промпта...", 0)

	genErrCh := make(chan error, 1)
	tokenCh := make(chan string)
	go func() {
		defer close(tokenCh)
		genErrCh <- m.llama.Generate(ctx, systemPrompt, ragPrompt, tokenCh)
	}()

	m.sendStage(reqID, "streaming", "Генерация ответа...", 0)

	if err := m.streaming.Stream(ctx, tokenCh, func(data []byte) error {
		return m.ipc.Send(ipc.NewMessage(ipc.TypeToken, reqID, data))
	}); err != nil {
		m.log.Error("streaming failed", "error", err)
	}

	if err := <-genErrCh; err != nil {
		m.sendError(fmt.Sprintf("generation failed: %v", err), reqID)
		return
	}

	m.sendSources(reqID, docs)
	m.ipc.Send(ipc.NewMessage(ipc.TypeDone, reqID, nil))
	m.log.Info("pipeline complete")
}

func (m *Manager) handleProfileList(msg *ipc.Message) {
	profiles := m.profiles.ListProfiles()
	data, err := json.Marshal(profiles)
	if err != nil {
		m.ipc.SendError(msg.Header.RequestID, jsonError("INTERNAL",
			fmt.Sprintf("marshal profiles: %v", err)))
		return
	}
	m.ipc.Send(ipc.NewMessage(ipc.TypeProfileList, msg.Header.RequestID, data))
}

func (m *Manager) handleProfileSwitch(msg *ipc.Message) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		m.ipc.SendError(msg.Header.RequestID, jsonError("BAD_REQUEST",
			fmt.Sprintf("invalid payload: %v", err)))
		return
	}

	m.mu.Lock()
	if m.active {
		m.mu.Unlock()
		m.ipc.SendError(msg.Header.RequestID, []byte(`{"code":"BUSY","message":"Cannot switch profile while request in progress"}`))
		return
	}
	err := m.profiles.SwitchTo(req.Name)
	m.mu.Unlock()

	if err != nil {
		m.ipc.SendError(msg.Header.RequestID, jsonError("NOT_FOUND",
			fmt.Sprintf("%v", err)))
		return
	}

	m.ipc.Send(ipc.NewMessage(ipc.TypePong, msg.Header.RequestID, []byte(`{"status":"ok"}`)))
}

type sourceDoc struct {
	Index int     `json:"index"`
	Title string  `json:"title"`
	Source string `json:"source"`
	Score  float64 `json:"score"`
}

func (m *Manager) sendStage(reqID uint32, stage, detail string, count int) {
	payload, err := json.Marshal(map[string]interface{}{
		"stage":  stage,
		"detail": detail,
		"count":  count,
	})
	if err != nil {
		return
	}
	m.ipc.Send(ipc.NewMessage(ipc.TypeStage, reqID, payload))
}

func (m *Manager) sendSources(reqID uint32, docs []pipeline.Document) {
	srcs := make([]sourceDoc, 0, len(docs))
	for _, d := range docs {
		srcs = append(srcs, sourceDoc{
			Index:  d.Index,
			Title:  d.Title,
			Source: d.Source,
			Score:  d.Score,
		})
	}
	data, err := json.Marshal(srcs)
	if err != nil {
		m.log.Error("marshal sources", "error", err)
		return
	}
	if err := m.ipc.Send(ipc.NewMessage(ipc.TypeSources, reqID, data)); err != nil {
		m.log.Error("send sources", "error", err)
	}
}

func (m *Manager) sendError(msg string, reqID uint32) {
	m.log.Error("pipeline error", "error", msg)
	m.ipc.SendError(reqID, jsonError("PIPELINE_ERROR", msg))
}