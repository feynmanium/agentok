package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	edqlite "github.com/dustland/agentok/edq-lite"
)

// Handler provides HTTP endpoints for the EDQ Lite broker.
type Handler struct {
	broker  *edqlite.Broker
	agents  map[string]edqlite.AgentMailbox
	agentMu sync.RWMutex
}

// NewHandler creates a new HTTP handler for the given broker.
func NewHandler(broker *edqlite.Broker) *Handler {
	return &Handler{
		broker: broker,
		agents: make(map[string]edqlite.AgentMailbox),
	}
}

// RegisterRoutes registers all API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/events", h.publishEvent)
	mux.HandleFunc("GET /api/v1/events/stream", h.streamEvents)
	mux.HandleFunc("GET /api/v1/topics", h.listTopics)
	mux.HandleFunc("POST /api/v1/agents", h.registerAgent)
	mux.HandleFunc("DELETE /api/v1/agents/{name}", h.deregisterAgent)
	mux.HandleFunc("POST /api/v1/agents/{name}/send", h.agentSend)
	mux.HandleFunc("GET /api/v1/agents/{name}/receive", h.agentReceive)
	mux.HandleFunc("GET /api/v1/health", h.healthCheck)
	mux.HandleFunc("GET /api/v1/stats", h.stats)
}

type publishRequest struct {
	Topic    string         `json:"topic"`
	Type     string         `json:"type"`
	Sender   string         `json:"sender"`
	Receiver string         `json:"receiver,omitempty"`
	Content  string         `json:"content"`
	Priority int            `json:"priority,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (h *Handler) publishEvent(w http.ResponseWriter, r *http.Request) {
	var req publishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	evt := edqlite.NewEvent(req.Topic, req.Type, req.Sender, req.Receiver, req.Content)
	if req.Priority != 0 {
		evt.Priority = edqlite.Priority(req.Priority)
	}
	evt.Metadata = req.Metadata

	if err := h.broker.Publish(r.Context(), evt); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"id": evt.ID, "status": "accepted"})
}

func (h *Handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	pattern := r.URL.Query().Get("topic")
	if pattern == "" {
		pattern = ">"
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	eventCh := make(chan edqlite.Event, 32)
	subID, err := h.broker.Subscribe(edqlite.TopicPattern(pattern), func(evt edqlite.Event) error {
		select {
		case eventCh <- evt:
			return nil
		default:
			return nil // drop if buffer full
		}
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer h.broker.Unsubscribe(subID)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-eventCh:
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "id: %s\ndata: %s\n\n", evt.ID, data)
			flusher.Flush()
		}
	}
}

func (h *Handler) listTopics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.broker.Topics())
}

type registerAgentRequest struct {
	Name      string `json:"name"`
	AgentType string `json:"agent_type"`
	ChatID    string `json:"chat_id,omitempty"`
}

func (h *Handler) registerAgent(w http.ResponseWriter, r *http.Request) {
	var req registerAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	h.agentMu.Lock()
	defer h.agentMu.Unlock()

	if _, exists := h.agents[req.Name]; exists {
		writeError(w, http.StatusConflict, "agent already registered")
		return
	}

	var agentOpts []edqlite.AgentOption
	if req.ChatID != "" {
		agentOpts = append(agentOpts, edqlite.WithAgentChatID(req.ChatID))
	}

	mb, err := edqlite.NewAgentMailbox(h.broker, req.Name, edqlite.AgentType(req.AgentType), agentOpts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.agents[req.Name] = mb

	writeJSON(w, http.StatusCreated, map[string]string{"name": req.Name, "status": "registered"})
}

func (h *Handler) deregisterAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	h.agentMu.Lock()
	defer h.agentMu.Unlock()

	mb, exists := h.agents[name]
	if !exists {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	_ = mb.Close()
	delete(h.agents, name)

	writeJSON(w, http.StatusOK, map[string]string{"name": name, "status": "deregistered"})
}

type agentSendRequest struct {
	Target   string         `json:"target,omitempty"`
	Content  string         `json:"content"`
	Type     string         `json:"type,omitempty"`
	Priority int            `json:"priority,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (h *Handler) agentSend(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	h.agentMu.RLock()
	mb, exists := h.agents[name]
	h.agentMu.RUnlock()

	if !exists {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	var req agentSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	var opts []edqlite.SendOption
	if req.Type != "" {
		opts = append(opts, edqlite.WithSendType(req.Type))
	}
	if req.Priority != 0 {
		opts = append(opts, edqlite.WithSendPriority(edqlite.Priority(req.Priority)))
	}
	if req.Metadata != nil {
		opts = append(opts, edqlite.WithSendMetadata(req.Metadata))
	}

	if err := mb.Send(r.Context(), req.Target, req.Content, opts...); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
}

func (h *Handler) agentReceive(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	h.agentMu.RLock()
	mb, exists := h.agents[name]
	h.agentMu.RUnlock()

	if !exists {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-mb.Receive():
			if !ok {
				return
			}
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "id: %s\ndata: %s\n\n", evt.ID, data)
			flusher.Flush()
		}
	}
}

func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.broker.Stats())
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// pathParam extracts a path parameter from a URL path.
// This is a fallback for pre-1.22 net/http without PathValue.
func pathParam(path, prefix string) string {
	path = strings.TrimPrefix(path, prefix)
	if idx := strings.Index(path, "/"); idx != -1 {
		return path[:idx]
	}
	return path
}

// Close cleans up all registered agent mailboxes.
func (h *Handler) Close() {
	h.agentMu.Lock()
	defer h.agentMu.Unlock()
	for name, mb := range h.agents {
		_ = mb.Close()
		delete(h.agents, name)
	}
}

// Ensure io is used (for potential future request body handling).
var _ = io.EOF
