package sdk

import "github.com/xEcho1337/fable/component"

// Result is the outcome returned by a plugin event handler.
func reasonString(reason component.Component) string {
	raw, err := component.Marshal(reason)
	if err != nil {
		return ""
	}
	if string(raw) == "null" {
		return ""
	}
	return string(raw)
}

// Registry collects handlers exposed by a plugin.
type Registry struct {
	login      []func(*LoginEvent)
	handshake  []func(*HandshakeEvent)
	disconnect []func(*DisconnectEvent)
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// OnLogin registers a handler for login events.
func (r *Registry) OnLogin(handler func(*LoginEvent)) {
	if handler != nil {
		r.login = append(r.login, handler)
	}
}

// OnHandshake registers a handler for handshake events.
func (r *Registry) OnHandshake(handler func(*HandshakeEvent)) {
	if handler != nil {
		r.handshake = append(r.handshake, handler)
	}
}

// OnDisconnect registers a handler for disconnect events.
func (r *Registry) OnDisconnect(handler func(*DisconnectEvent)) {
	if handler != nil {
		r.disconnect = append(r.disconnect, handler)
	}
}

// Register attaches all plugin handlers to a Relay event bus.
func (r *Registry) Register(bus *Bus) {
	for _, handler := range r.login {
		bus.Login.Subscribe(func(event LoginEvent) (Decision, string) {
			handler(&event)
			decision, reason := event.Outcome()
			return decision, reasonString(reason)
		})
	}
	for _, handler := range r.handshake {
		bus.Handshake.Subscribe(func(event HandshakeEvent) (Decision, string) {
			handler(&event)
			decision, reason := event.Outcome()
			return decision, reasonString(reason)
		})
	}
	for _, handler := range r.disconnect {
		bus.Disconnect.Subscribe(func(event DisconnectEvent) (Decision, string) {
			handler(&event)
			decision, reason := event.Outcome()
			return decision, reasonString(reason)
		})
	}
}
