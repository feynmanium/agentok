// Package edqlite implements EDQ Lite (Event-Driven Queue Lite), a lightweight,
// pure-Go event-driven message queue designed for multi-agent communication
// in the Agentok Studio platform.
//
// # Software Design Document
//
// ## Purpose and Scope
//
// EDQ Lite provides an embeddable, zero-dependency event-driven message queue
// that models Agentok Studio's agent communication patterns. It is designed to
// be used either as an in-process library or as a standalone service via its
// HTTP API.
//
// ## Architecture Overview
//
// The system is composed of six core components:
//
//   - Event: The fundamental unit of communication, carrying sender/receiver
//     metadata, typed content, priority, and arbitrary metadata. Maps directly
//     to Agentok's Message model.
//
//   - Topic: A hierarchical, dot-separated namespace for routing events.
//     Supports NATS-style wildcards: "*" matches a single segment, ">" matches
//     one or more trailing segments.
//
//   - Subscription: Binds a handler function to a topic pattern with an
//     optional filter predicate for fine-grained event selection.
//
//   - Broker: The central dispatch engine. Accepts published events into a
//     priority queue, dispatches them via a pool of worker goroutines to
//     matching subscriptions. Thread-safe for concurrent use.
//
//   - AgentMailbox: A higher-level abstraction providing send/receive semantics
//     for individual agents, aligned with Agentok's sender/receiver model.
//
//   - TopicRouter: A rule-based routing table that maps agent-to-agent
//     communication edges (as defined in Agentok flow graphs) to topics.
//
// ## Topic Naming Convention
//
//	chat.{chatID}.agent.{name}     -- direct messages to a specific agent
//	chat.{chatID}.broadcast        -- broadcast to all agents in a chat
//	chat.{chatID}.system           -- system events (status, errors)
//	chat.{chatID}.group.{name}     -- group chat messages
//	agent.{name}.>                 -- all events involving a specific agent
//
// ## Concurrency Model
//
// The Broker uses a buffered channel as the ingest point for published events.
// A configurable number of worker goroutines drain this channel, insert events
// into a heap-based priority queue, and fan out to matching subscriptions.
// The subscription registry is protected by sync.RWMutex to allow concurrent
// reads during dispatch while serializing subscription mutations.
//
// ## Priority Dispatch
//
// Events are ordered by priority (descending) then by creation time (ascending,
// FIFO within the same priority tier). Four priority levels are defined:
// Low (0), Normal (5), High (10), and Urgent (15).
//
// ## HTTP API
//
// The optional HTTP layer exposes:
//
//	POST   /api/v1/events              -- publish an event
//	GET    /api/v1/events/stream       -- SSE stream filtered by topic pattern
//	GET    /api/v1/topics              -- list active topics with subscriber counts
//	POST   /api/v1/agents              -- register an agent mailbox
//	DELETE /api/v1/agents/{name}       -- deregister an agent
//	POST   /api/v1/agents/{name}/send  -- send from an agent
//	GET    /api/v1/agents/{name}/receive -- SSE stream for an agent
//	GET    /api/v1/health              -- health check
//
// ## Integration with Agentok
//
// Event.Sender and Event.Receiver map to Agentok flow node names. Event.Type
// mirrors Agentok message types (user, assistant, summary, system, tool_call,
// tool_response). Chat IDs scope topics to individual conversation sessions.
// The TopicRouter can be configured from an Agentok flow graph's edge list to
// automatically route agent-to-agent messages.
package edqlite
