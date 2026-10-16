package sdk

import "sync"

// EventHandler receives an event and returns the resulting decision and payload.
type EventHandler[T any] func(T) (Decision, string)

// Topic stores subscribers for one event type and emits events safely.
type Topic[T any] struct {
	mu       sync.RWMutex
	handlers []EventHandler[T]
}

// Subscribe adds a handler to the topic. Handlers may return a decision and
// payload; notification-only handlers are also accepted for events that do not
// use a decision. Nil and unsupported handlers are ignored.
func (t *Topic[T]) Subscribe(handler any) {
	var eventHandler EventHandler[T]
	switch typedHandler := handler.(type) {
	case EventHandler[T]:
		eventHandler = typedHandler
	case func(T) (Decision, string):
		eventHandler = typedHandler
	case func(T):
		eventHandler = func(event T) (Decision, string) {
			typedHandler(event)
			return Allow, ""
		}
	default:
		return
	}

	if eventHandler == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers = append(t.handlers, eventHandler)
}

// Emit notifies handlers in registration order and returns the resulting decision.
// Deny stops delivery immediately; the last OverrideBackend result is retained.
func (t *Topic[T]) Emit(event T) (Decision, string) {
	t.mu.RLock()
	handlers := make([]EventHandler[T], len(t.handlers))
	copy(handlers, t.handlers)
	t.mu.RUnlock()

	decision := Allow
	payload := ""
	for _, handler := range handlers {
		switch currentDecision, currentPayload := handler(event); currentDecision {
		case Deny:
			return Deny, currentPayload
		case OverrideBackend:
			decision = OverrideBackend
			payload = currentPayload
		}
	}

	return decision, payload
}

type HandshakeHandler = EventHandler[HandshakeEvent]
type LoginHandler = EventHandler[LoginEvent]
type DisconnectHandler = EventHandler[DisconnectEvent]

// Bus exposes one generic topic for each supported event type.
type Bus struct {
	Handshake  Topic[HandshakeEvent]
	Login      Topic[LoginEvent]
	Disconnect Topic[DisconnectEvent]
}

// NewBus creates an empty event bus.
func NewBus() *Bus {
	return &Bus{}
}

var DefaultBus = NewBus()
