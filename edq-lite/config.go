package edqlite

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the full YAML configuration for an EDQ Lite instance.
type Config struct {
	Broker  BrokerConfig  `yaml:"broker"`
	Agents  []AgentConfig `yaml:"agents,omitempty"`
	Routes  []RouteConfig `yaml:"routes,omitempty"`
	Rules   []RuleConfig  `yaml:"rules,omitempty"`
	HTTP    HTTPConfig    `yaml:"http,omitempty"`
}

// BrokerConfig is the YAML broker section.
type BrokerConfig struct {
	QueueDepth int `yaml:"queue_depth,omitempty"`
	Workers    int `yaml:"workers,omitempty"`
}

// AgentConfig is the YAML agent section.
type AgentConfig struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	ChatID     string `yaml:"chat_id,omitempty"`
	BufferSize int    `yaml:"buffer_size,omitempty"`
}

// RouteConfig is the YAML route section.
type RouteConfig struct {
	Source    string `yaml:"source"`
	Target   string `yaml:"target,omitempty"`
	Topic    string `yaml:"topic"`
	Broadcast bool  `yaml:"broadcast,omitempty"`
}

// RuleConfig is the YAML rule section.
type RuleConfig struct {
	Name           string   `yaml:"name"`
	Type           string   `yaml:"type"` // "filter_by_type", "filter_by_sender", "max_content_length", "priority_boost", "metadata_enrich", "conditional_route"
	AllowedTypes   []string `yaml:"allowed_types,omitempty"`
	AllowedSenders []string `yaml:"allowed_senders,omitempty"`
	MaxLength      int      `yaml:"max_length,omitempty"`
	BoostPriority  int      `yaml:"boost_priority,omitempty"`
	BoostTypes     []string `yaml:"boost_types,omitempty"`
	Enrichments    map[string]any `yaml:"enrichments,omitempty"`
	RouteTo        string   `yaml:"route_to,omitempty"`
	RouteTypes     []string `yaml:"route_types,omitempty"`
}

// HTTPConfig is the YAML HTTP server section.
type HTTPConfig struct {
	Addr string `yaml:"addr,omitempty"`
}

// CircuitBreakerYAMLConfig is the YAML circuit breaker section.
type CircuitBreakerYAMLConfig struct {
	MaxFailures         int    `yaml:"max_failures,omitempty"`
	ResetTimeoutSeconds int    `yaml:"reset_timeout_seconds,omitempty"`
	HalfOpenMaxAttempts int    `yaml:"half_open_max_attempts,omitempty"`
}

// LoadConfig reads and parses a YAML configuration file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	return ParseConfig(data)
}

// ParseConfig parses YAML data into a Config.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if c.Broker.QueueDepth < 0 {
		return fmt.Errorf("config: broker.queue_depth must be >= 0")
	}
	if c.Broker.Workers < 0 {
		return fmt.Errorf("config: broker.workers must be >= 0")
	}

	agentNames := make(map[string]bool)
	for i, a := range c.Agents {
		if a.Name == "" {
			return fmt.Errorf("config: agents[%d].name is required", i)
		}
		if agentNames[a.Name] {
			return fmt.Errorf("config: duplicate agent name %q", a.Name)
		}
		agentNames[a.Name] = true
	}

	for i, r := range c.Routes {
		if r.Source == "" {
			return fmt.Errorf("config: routes[%d].source is required", i)
		}
		if r.Topic == "" {
			return fmt.Errorf("config: routes[%d].topic is required", i)
		}
		if !r.Broadcast && r.Target == "" {
			return fmt.Errorf("config: routes[%d].target is required when not broadcast", i)
		}
	}

	ruleNames := make(map[string]bool)
	for i, r := range c.Rules {
		if r.Name == "" {
			return fmt.Errorf("config: rules[%d].name is required", i)
		}
		if ruleNames[r.Name] {
			return fmt.Errorf("config: duplicate rule name %q", r.Name)
		}
		ruleNames[r.Name] = true
		if r.Type == "" {
			return fmt.Errorf("config: rules[%d].type is required", i)
		}
	}

	return nil
}

// BuildBrokerOptions converts the config into BrokerOption slice.
func (c *Config) BuildBrokerOptions() []BrokerOption {
	var opts []BrokerOption
	if c.Broker.QueueDepth > 0 {
		opts = append(opts, WithQueueDepth(c.Broker.QueueDepth))
	}
	if c.Broker.Workers > 0 {
		opts = append(opts, WithWorkers(c.Broker.Workers))
	}
	return opts
}

// BuildRuleChain creates a RuleChain from the rules config.
func (c *Config) BuildRuleChain() (*RuleChain, error) {
	chain := NewRuleChain()
	for _, rc := range c.Rules {
		rule, err := buildRuleFromConfig(rc)
		if err != nil {
			return nil, err
		}
		chain.AddRule(rule)
	}
	return chain, nil
}

// BuildRouter configures a TopicRouter from the routes config.
func (c *Config) BuildRouter(broker *Broker) (*TopicRouter, error) {
	router := NewTopicRouter(broker)
	for _, r := range c.Routes {
		if r.Broadcast {
			if err := router.AddBroadcastRoute(r.Source, r.Topic); err != nil {
				return nil, fmt.Errorf("config: route %s->broadcast: %w", r.Source, err)
			}
		} else {
			if err := router.AddRoute(r.Source, r.Target, r.Topic); err != nil {
				return nil, fmt.Errorf("config: route %s->%s: %w", r.Source, r.Target, err)
			}
		}
	}
	return router, nil
}

// BuildAgents creates agent mailboxes from the agents config.
func (c *Config) BuildAgents(broker *Broker) (map[string]AgentMailbox, error) {
	agents := make(map[string]AgentMailbox, len(c.Agents))
	for _, ac := range c.Agents {
		var opts []AgentOption
		if ac.ChatID != "" {
			opts = append(opts, WithAgentChatID(ac.ChatID))
		}
		if ac.BufferSize > 0 {
			opts = append(opts, WithBufferSize(ac.BufferSize))
		}
		mb, err := NewAgentMailbox(broker, ac.Name, AgentType(ac.Type), opts...)
		if err != nil {
			// Cleanup on failure
			for _, a := range agents {
				a.Close()
			}
			return nil, fmt.Errorf("config: agent %s: %w", ac.Name, err)
		}
		agents[ac.Name] = mb
	}
	return agents, nil
}

// MarshalConfig serializes a Config to YAML bytes.
func MarshalConfig(cfg *Config) ([]byte, error) {
	return yaml.Marshal(cfg)
}

func buildRuleFromConfig(rc RuleConfig) (Rule, error) {
	switch rc.Type {
	case "filter_by_type":
		if len(rc.AllowedTypes) == 0 {
			return nil, fmt.Errorf("config: rule %q: filter_by_type requires allowed_types", rc.Name)
		}
		return FilterByTypeRule(rc.AllowedTypes...), nil

	case "filter_by_sender":
		if len(rc.AllowedSenders) == 0 {
			return nil, fmt.Errorf("config: rule %q: filter_by_sender requires allowed_senders", rc.Name)
		}
		return FilterBySenderRule(rc.AllowedSenders...), nil

	case "max_content_length":
		if rc.MaxLength <= 0 {
			return nil, fmt.Errorf("config: rule %q: max_content_length requires max_length > 0", rc.Name)
		}
		return MaxContentLengthRule(rc.MaxLength), nil

	case "priority_boost":
		if rc.BoostPriority <= 0 {
			return nil, fmt.Errorf("config: rule %q: priority_boost requires boost_priority > 0", rc.Name)
		}
		allowedTypes := make(map[string]bool, len(rc.BoostTypes))
		for _, t := range rc.BoostTypes {
			allowedTypes[t] = true
		}
		return PriorityBoostRule(rc.Name, func(evt Event) bool {
			if len(allowedTypes) == 0 {
				return true
			}
			return allowedTypes[evt.Type]
		}, Priority(rc.BoostPriority)), nil

	case "metadata_enrich":
		if len(rc.Enrichments) == 0 {
			return nil, fmt.Errorf("config: rule %q: metadata_enrich requires enrichments", rc.Name)
		}
		return MetadataEnrichRule(rc.Name, rc.Enrichments), nil

	case "conditional_route":
		if rc.RouteTo == "" {
			return nil, fmt.Errorf("config: rule %q: conditional_route requires route_to", rc.Name)
		}
		allowedTypes := make(map[string]bool, len(rc.RouteTypes))
		for _, t := range rc.RouteTypes {
			allowedTypes[t] = true
		}
		return ConditionalRouteRule(rc.Name, func(evt Event) bool {
			if len(allowedTypes) == 0 {
				return true
			}
			return allowedTypes[evt.Type]
		}, rc.RouteTo), nil

	default:
		return nil, fmt.Errorf("config: rule %q: unknown type %q", rc.Name, rc.Type)
	}
}
