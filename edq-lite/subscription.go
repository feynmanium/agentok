package edqlite

// HandlerFunc processes a received event. Returning an error signals the broker
// to record a delivery failure.
type HandlerFunc func(evt Event) error

// Filter is an optional predicate applied before dispatch.
type Filter func(evt Event) bool

// Subscription binds a handler to a topic pattern.
type Subscription struct {
	ID      string
	Pattern TopicPattern
	Filter  Filter      // nil means accept all
	Handler HandlerFunc
}

// matches checks if the subscription matches the event.
func (s *Subscription) matches(evt Event) bool {
	if !s.Pattern.Match(evt.Topic) {
		return false
	}
	if s.Filter != nil && !s.Filter(evt) {
		return false
	}
	return true
}
