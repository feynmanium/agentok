package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	edqlite "github.com/dustland/agentok/edq-lite"
)

func setupTestHandler(t *testing.T) (*Handler, *edqlite.Broker, func()) {
	t.Helper()
	b := edqlite.NewBroker(edqlite.WithWorkers(2), edqlite.WithQueueDepth(64))
	ctx := context.Background()
	if err := b.Start(ctx); err != nil {
		t.Fatalf("broker start: %v", err)
	}
	h := NewHandler(b)
	cleanup := func() {
		h.Close()
		b.Shutdown(ctx)
	}
	return h, b, cleanup
}

func TestHealthCheck(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

func TestPublishEvent(t *testing.T) {
	h, b, cleanup := setupTestHandler(t)
	defer cleanup()

	received := make(chan edqlite.Event, 1)
	b.Subscribe("chat.1.agent.bob", func(evt edqlite.Event) error {
		received <- evt
		return nil
	})

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := `{"topic":"chat.1.agent.bob","type":"user","sender":"alice","receiver":"bob","content":"hello"}`
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	select {
	case evt := <-received:
		if evt.Content != "hello" {
			t.Errorf("content = %q, want %q", evt.Content, "hello")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestPublishEventInvalidBody(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListTopics(t *testing.T) {
	h, b, cleanup := setupTestHandler(t)
	defer cleanup()

	b.Subscribe("chat.1.agent.*", func(edqlite.Event) error { return nil })
	b.Subscribe("chat.1.broadcast", func(edqlite.Event) error { return nil })

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/topics", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var topics []edqlite.TopicInfo
	json.NewDecoder(w.Body).Decode(&topics)
	if len(topics) != 2 {
		t.Errorf("got %d topics, want 2", len(topics))
	}
}

func TestRegisterAndDeregisterAgent(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Register
	body := `{"name":"test-agent","agent_type":"AssistantAgent","chat_id":"1"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("register status = %d, want %d", w.Code, http.StatusCreated)
	}

	// Duplicate register
	req2 := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewBufferString(body))
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Errorf("duplicate register status = %d, want %d", w2.Code, http.StatusConflict)
	}

	// Deregister
	req3 := httptest.NewRequest("DELETE", "/api/v1/agents/test-agent", nil)
	w3 := httptest.NewRecorder()
	mux.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("deregister status = %d, want %d", w3.Code, http.StatusOK)
	}

	// Deregister non-existent
	req4 := httptest.NewRequest("DELETE", "/api/v1/agents/nonexistent", nil)
	w4 := httptest.NewRecorder()
	mux.ServeHTTP(w4, req4)
	if w4.Code != http.StatusNotFound {
		t.Errorf("deregister missing status = %d, want %d", w4.Code, http.StatusNotFound)
	}
}

func TestAgentSend(t *testing.T) {
	h, b, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Register sender agent
	regBody := `{"name":"sender","agent_type":"UserProxyAgent","chat_id":"1"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewBufferString(regBody))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Subscribe to receive
	received := make(chan edqlite.Event, 1)
	b.Subscribe("chat.1.agent.receiver", func(evt edqlite.Event) error {
		received <- evt
		return nil
	})

	// Send
	sendBody := `{"target":"receiver","content":"hello from api","type":"user"}`
	req2 := httptest.NewRequest("POST", "/api/v1/agents/sender/send", bytes.NewBufferString(sendBody))
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusAccepted {
		t.Errorf("send status = %d, want %d", w2.Code, http.StatusAccepted)
	}

	select {
	case evt := <-received:
		if evt.Content != "hello from api" {
			t.Errorf("content = %q, want %q", evt.Content, "hello from api")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestAgentSendNotFound(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := `{"target":"bob","content":"hi"}`
	req := httptest.NewRequest("POST", "/api/v1/agents/nonexistent/send", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestStats(t *testing.T) {
	h, b, cleanup := setupTestHandler(t)
	defer cleanup()

	b.Subscribe("test.>", func(edqlite.Event) error { return nil })

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Publish an event
	body := `{"topic":"test.stats","type":"system","sender":"sys","content":"test"}`
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	time.Sleep(200 * time.Millisecond)

	// Check stats
	req2 := httptest.NewRequest("GET", "/api/v1/stats", nil)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusOK)
	}

	var stats edqlite.BrokerStats
	json.NewDecoder(w2.Body).Decode(&stats)
	if stats.Published < 1 {
		t.Errorf("published = %d, want >= 1", stats.Published)
	}
}

func TestRegisterAgentMissingName(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := `{"agent_type":"AssistantAgent"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
