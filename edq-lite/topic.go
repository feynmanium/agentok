package edqlite

import (
	"errors"
	"strings"
)

// TopicPattern represents a subscription pattern with optional wildcards.
// Supports:
//   - Exact: "chat.42.agent.assistant_1"
//   - Single wildcard: "chat.42.agent.*"   (matches exactly one segment)
//   - Multi wildcard:  "chat.42.>"         (matches one or more trailing segments)
type TopicPattern string

// Match reports whether the pattern matches the given topic string.
func (p TopicPattern) Match(topic string) bool {
	patternParts := strings.Split(string(p), ".")
	topicParts := strings.Split(topic, ".")

	return matchParts(patternParts, topicParts)
}

func matchParts(pattern, topic []string) bool {
	pi, ti := 0, 0
	for pi < len(pattern) && ti < len(topic) {
		seg := pattern[pi]
		switch seg {
		case ">":
			// ">" must be the last segment and matches one or more remaining segments.
			return pi == len(pattern)-1
		case "*":
			// "*" matches exactly one segment.
			pi++
			ti++
		default:
			if seg != topic[ti] {
				return false
			}
			pi++
			ti++
		}
	}
	// Both must be fully consumed (unless pattern ended with ">").
	return pi == len(pattern) && ti == len(topic)
}

// ValidateTopic checks that a concrete topic string is well-formed.
// Concrete topics must not contain wildcards (* or >).
func ValidateTopic(topic string) error {
	if topic == "" {
		return errors.New("topic: must not be empty")
	}
	parts := strings.Split(topic, ".")
	for _, part := range parts {
		if part == "" {
			return errors.New("topic: must not contain empty segments")
		}
		if part == "*" || part == ">" {
			return errors.New("topic: concrete topics must not contain wildcards")
		}
	}
	return nil
}

// ValidatePattern checks that a topic pattern is well-formed.
func ValidatePattern(pattern string) error {
	if pattern == "" {
		return errors.New("pattern: must not be empty")
	}
	parts := strings.Split(pattern, ".")
	for i, part := range parts {
		if part == "" {
			return errors.New("pattern: must not contain empty segments")
		}
		if part == ">" && i != len(parts)-1 {
			return errors.New("pattern: '>' must be the last segment")
		}
	}
	return nil
}
