package edqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const validYAML = `
broker:
  queue_depth: 256
  workers: 4

agents:
  - name: alice
    type: UserProxyAgent
    chat_id: "42"
    buffer_size: 32
  - name: bob
    type: AssistantAgent
    chat_id: "42"

routes:
  - source: alice
    target: bob
    topic: chat.42.agent.bob
  - source: alice
    topic: chat.42.broadcast
    broadcast: true

rules:
  - name: only-user-messages
    type: filter_by_type
    allowed_types:
      - user
      - assistant
  - name: max-size
    type: max_content_length
    max_length: 1000
  - name: enrich-env
    type: metadata_enrich
    enrichments:
      env: production
      version: "1.0"

http:
  addr: ":9000"
`

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig([]byte(validYAML))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	// Broker
	if cfg.Broker.QueueDepth != 256 {
		t.Errorf("broker.queue_depth = %d, want 256", cfg.Broker.QueueDepth)
	}
	if cfg.Broker.Workers != 4 {
		t.Errorf("broker.workers = %d, want 4", cfg.Broker.Workers)
	}

	// Agents
	if len(cfg.Agents) != 2 {
		t.Fatalf("agents count = %d, want 2", len(cfg.Agents))
	}
	if cfg.Agents[0].Name != "alice" {
		t.Errorf("agents[0].name = %q, want %q", cfg.Agents[0].Name, "alice")
	}
	if cfg.Agents[0].Type != "UserProxyAgent" {
		t.Errorf("agents[0].type = %q, want %q", cfg.Agents[0].Type, "UserProxyAgent")
	}
	if cfg.Agents[0].BufferSize != 32 {
		t.Errorf("agents[0].buffer_size = %d, want 32", cfg.Agents[0].BufferSize)
	}

	// Routes
	if len(cfg.Routes) != 2 {
		t.Fatalf("routes count = %d, want 2", len(cfg.Routes))
	}
	if !cfg.Routes[1].Broadcast {
		t.Error("routes[1] should be broadcast")
	}

	// Rules
	if len(cfg.Rules) != 3 {
		t.Fatalf("rules count = %d, want 3", len(cfg.Rules))
	}
	if cfg.Rules[0].Type != "filter_by_type" {
		t.Errorf("rules[0].type = %q", cfg.Rules[0].Type)
	}

	// HTTP
	if cfg.HTTP.Addr != ":9000" {
		t.Errorf("http.addr = %q, want %q", cfg.HTTP.Addr, ":9000")
	}
}

func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(validYAML), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Broker.Workers != 4 {
		t.Errorf("workers = %d, want 4", cfg.Broker.Workers)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParseConfigInvalidYAML(t *testing.T) {
	_, err := ParseConfig([]byte("not: valid: yaml: ["))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name:    "valid minimal",
			yaml:    "broker:\n  workers: 1",
			wantErr: false,
		},
		{
			name:    "negative queue depth",
			yaml:    "broker:\n  queue_depth: -1",
			wantErr: true,
		},
		{
			name:    "negative workers",
			yaml:    "broker:\n  workers: -1",
			wantErr: true,
		},
		{
			name:    "agent missing name",
			yaml:    "broker: {}\nagents:\n  - type: AssistantAgent",
			wantErr: true,
		},
		{
			name:    "duplicate agent name",
			yaml:    "broker: {}\nagents:\n  - name: a\n    type: X\n  - name: a\n    type: Y",
			wantErr: true,
		},
		{
			name:    "route missing source",
			yaml:    "broker: {}\nroutes:\n  - target: b\n    topic: t.t",
			wantErr: true,
		},
		{
			name:    "route missing topic",
			yaml:    "broker: {}\nroutes:\n  - source: a\n    target: b",
			wantErr: true,
		},
		{
			name:    "non-broadcast route missing target",
			yaml:    "broker: {}\nroutes:\n  - source: a\n    topic: t.t",
			wantErr: true,
		},
		{
			name:    "broadcast route without target is ok",
			yaml:    "broker: {}\nroutes:\n  - source: a\n    topic: t.t\n    broadcast: true",
			wantErr: false,
		},
		{
			name:    "rule missing name",
			yaml:    "broker: {}\nrules:\n  - type: filter_by_type\n    allowed_types: [user]",
			wantErr: true,
		},
		{
			name:    "rule missing type",
			yaml:    "broker: {}\nrules:\n  - name: r1",
			wantErr: true,
		},
		{
			name:    "duplicate rule name",
			yaml:    "broker: {}\nrules:\n  - name: r1\n    type: filter_by_type\n    allowed_types: [user]\n  - name: r1\n    type: max_content_length\n    max_length: 10",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildBrokerOptions(t *testing.T) {
	cfg, _ := ParseConfig([]byte(validYAML))
	opts := cfg.BuildBrokerOptions()
	if len(opts) != 2 {
		t.Errorf("options count = %d, want 2", len(opts))
	}
}

func TestBuildBrokerOptionsDefaults(t *testing.T) {
	cfg, _ := ParseConfig([]byte("broker: {}"))
	opts := cfg.BuildBrokerOptions()
	if len(opts) != 0 {
		t.Errorf("options count = %d, want 0 for defaults", len(opts))
	}
}

func TestBuildRuleChain(t *testing.T) {
	cfg, _ := ParseConfig([]byte(validYAML))
	chain, err := cfg.BuildRuleChain()
	if err != nil {
		t.Fatalf("BuildRuleChain() error = %v", err)
	}

	rules := chain.Rules()
	if len(rules) != 3 {
		t.Errorf("rules count = %d, want 3", len(rules))
	}

	// Test that the chain actually works
	ctx := context.Background()

	// User message should pass
	evt := NewEvent("t.a", "user", "alice", "bob", "hi")
	result := chain.Process(ctx, evt)
	if result.Action != RulePass {
		t.Errorf("user message should pass, got %v", result.Action)
	}
	if result.Event.Metadata["env"] != "production" {
		t.Error("metadata should be enriched")
	}

	// System message should be dropped
	evt2 := NewEvent("t.a", "system", "sys", "", "ping")
	result2 := chain.Process(ctx, evt2)
	if result2.Action != RuleDrop {
		t.Errorf("system message should be dropped, got %v", result2.Action)
	}
}

func TestBuildRuleChainAllTypes(t *testing.T) {
	yamlData := `
broker: {}
rules:
  - name: filter-type
    type: filter_by_type
    allowed_types: [user]
  - name: filter-sender
    type: filter_by_sender
    allowed_senders: [alice]
  - name: max-len
    type: max_content_length
    max_length: 100
  - name: boost
    type: priority_boost
    boost_priority: 15
    boost_types: [urgent]
  - name: enrich
    type: metadata_enrich
    enrichments:
      key: value
  - name: reroute
    type: conditional_route
    route_to: other.topic
    route_types: [system]
`
	cfg, err := ParseConfig([]byte(yamlData))
	if err != nil {
		t.Fatalf("ParseConfig error = %v", err)
	}

	chain, err := cfg.BuildRuleChain()
	if err != nil {
		t.Fatalf("BuildRuleChain error = %v", err)
	}

	if len(chain.Rules()) != 6 {
		t.Errorf("rules count = %d, want 6", len(chain.Rules()))
	}
}

func TestBuildRuleChainErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "filter_by_type missing allowed_types",
			yaml: "broker: {}\nrules:\n  - name: r1\n    type: filter_by_type",
		},
		{
			name: "filter_by_sender missing allowed_senders",
			yaml: "broker: {}\nrules:\n  - name: r1\n    type: filter_by_sender",
		},
		{
			name: "max_content_length missing max_length",
			yaml: "broker: {}\nrules:\n  - name: r1\n    type: max_content_length",
		},
		{
			name: "priority_boost missing boost_priority",
			yaml: "broker: {}\nrules:\n  - name: r1\n    type: priority_boost",
		},
		{
			name: "metadata_enrich missing enrichments",
			yaml: "broker: {}\nrules:\n  - name: r1\n    type: metadata_enrich",
		},
		{
			name: "conditional_route missing route_to",
			yaml: "broker: {}\nrules:\n  - name: r1\n    type: conditional_route",
		},
		{
			name: "unknown type",
			yaml: "broker: {}\nrules:\n  - name: r1\n    type: unknown_rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig([]byte(tt.yaml))
			if err != nil {
				t.Skipf("parse error (validation): %v", err)
			}
			_, err = cfg.BuildRuleChain()
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestBuildRouter(t *testing.T) {
	cfg, _ := ParseConfig([]byte(validYAML))
	b := NewBroker()
	router, err := cfg.BuildRouter(b)
	if err != nil {
		t.Fatalf("BuildRouter() error = %v", err)
	}

	// Direct route
	evt := Event{Sender: "alice", Receiver: "bob"}
	topic := router.RouteEvent(evt)
	if topic != "chat.42.agent.bob" {
		t.Errorf("direct route = %q, want %q", topic, "chat.42.agent.bob")
	}

	// Broadcast route
	evt2 := Event{Sender: "alice"}
	topic2 := router.RouteEvent(evt2)
	if topic2 != "chat.42.broadcast" {
		t.Errorf("broadcast route = %q, want %q", topic2, "chat.42.broadcast")
	}
}

func TestBuildAgents(t *testing.T) {
	cfg, _ := ParseConfig([]byte(validYAML))
	b := NewBroker()
	agents, err := cfg.BuildAgents(b)
	if err != nil {
		t.Fatalf("BuildAgents() error = %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("agents count = %d, want 2", len(agents))
	}

	alice, ok := agents["alice"]
	if !ok {
		t.Fatal("alice not found")
	}
	if alice.Name() != "alice" {
		t.Errorf("alice.Name() = %q", alice.Name())
	}
	if alice.Type() != AgentTypeUserProxy {
		t.Errorf("alice.Type() = %q, want %q", alice.Type(), AgentTypeUserProxy)
	}

	// Cleanup
	for _, a := range agents {
		a.Close()
	}
}

func TestMarshalConfig(t *testing.T) {
	cfg := &Config{
		Broker: BrokerConfig{QueueDepth: 128, Workers: 2},
		Agents: []AgentConfig{
			{Name: "test", Type: "AssistantAgent", ChatID: "1"},
		},
		HTTP: HTTPConfig{Addr: ":8900"},
	}

	data, err := MarshalConfig(cfg)
	if err != nil {
		t.Fatalf("MarshalConfig() error = %v", err)
	}

	// Round-trip
	cfg2, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("ParseConfig(marshaled) error = %v", err)
	}

	if cfg2.Broker.QueueDepth != 128 {
		t.Errorf("round-trip queue_depth = %d, want 128", cfg2.Broker.QueueDepth)
	}
	if len(cfg2.Agents) != 1 {
		t.Errorf("round-trip agents count = %d, want 1", len(cfg2.Agents))
	}
}

func TestBuildRuleChainPriorityBoostAllTypes(t *testing.T) {
	yaml := `
broker: {}
rules:
  - name: boost-all
    type: priority_boost
    boost_priority: 10
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseConfig error = %v", err)
	}
	chain, err := cfg.BuildRuleChain()
	if err != nil {
		t.Fatalf("BuildRuleChain error = %v", err)
	}

	ctx := context.Background()
	evt := NewEvent("t.a", "user", "alice", "bob", "hi")
	result := chain.Process(ctx, evt)
	if result.Event.Priority != PriorityHigh {
		t.Errorf("priority = %d, want %d", result.Event.Priority, PriorityHigh)
	}
}

func TestBuildRuleChainConditionalRouteAllTypes(t *testing.T) {
	yaml := `
broker: {}
rules:
  - name: route-all
    type: conditional_route
    route_to: catch.all.topic
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseConfig error = %v", err)
	}
	chain, err := cfg.BuildRuleChain()
	if err != nil {
		t.Fatalf("BuildRuleChain error = %v", err)
	}

	ctx := context.Background()
	evt := NewEvent("t.a", "anything", "alice", "bob", "hi")
	result := chain.Process(ctx, evt)
	if result.Action != RuleRoute {
		t.Errorf("action = %v, want route", result.Action)
	}
	if result.RouteTo != "catch.all.topic" {
		t.Errorf("route_to = %q, want %q", result.RouteTo, "catch.all.topic")
	}
}

func TestFullConfigIntegration(t *testing.T) {
	cfg, err := ParseConfig([]byte(validYAML))
	if err != nil {
		t.Fatalf("ParseConfig error = %v", err)
	}

	ctx := context.Background()

	// Build everything from config
	opts := cfg.BuildBrokerOptions()
	broker := NewBroker(opts...)

	router, err := cfg.BuildRouter(broker)
	if err != nil {
		t.Fatalf("BuildRouter error = %v", err)
	}

	agents, err := cfg.BuildAgents(broker)
	if err != nil {
		t.Fatalf("BuildAgents error = %v", err)
	}
	defer func() {
		for _, a := range agents {
			a.Close()
		}
	}()

	chain, err := cfg.BuildRuleChain()
	if err != nil {
		t.Fatalf("BuildRuleChain error = %v", err)
	}

	broker.Start(ctx)
	defer broker.Shutdown(ctx)

	// Process an event through rules then route and publish
	evt := NewEvent("", "user", "alice", "bob", "config integration test")
	result := chain.Process(ctx, evt)
	if result.Action != RulePass {
		t.Fatalf("rule chain should pass, got %v", result.Action)
	}

	result.Event.Topic = router.RouteEvent(result.Event)
	broker.Publish(ctx, result.Event)

	// Bob should receive via his mailbox
	bob := agents["bob"]
	select {
	case got := <-bob.Receive():
		if got.Content != "config integration test" {
			t.Errorf("content = %q", got.Content)
		}
		if got.Metadata["env"] != "production" {
			t.Error("metadata enrichment missing")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}
