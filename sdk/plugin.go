package sdk

// Result is the outcome returned by a plugin event handler.
type Result struct {
	Decision Decision
	Payload  string
}

// Registry collects handlers exposed by a plugin.
type Registry struct {
	login      []func(LoginEvent) Result
	handshake  []func(HandshakeEvent) Result
	disconnect []func(DisconnectEvent) Result
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// OnLogin registers a handler for login events.
func (r *Registry) OnLogin(handler func(LoginEvent) Result) {
	if handler != nil {
		r.login = append(r.login, handler)
	}
}

// OnHandshake registers a handler for handshake events.
func (r *Registry) OnHandshake(handler func(HandshakeEvent) Result) {
	if handler != nil {
		r.handshake = append(r.handshake, handler)
	}
}

// OnDisconnect registers a handler for disconnect events.
func (r *Registry) OnDisconnect(handler func(DisconnectEvent) Result) {
	if handler != nil {
		r.disconnect = append(r.disconnect, handler)
	}
}

// Register attaches all plugin handlers to a Relay event bus.
func (r *Registry) Register(bus *Bus) {
	for _, handler := range r.login {
		bus.Login.Subscribe(func(event LoginEvent) (Decision, string) {
			result := handler(event)
			return result.Decision, result.Payload
		})
	}
	for _, handler := range r.handshake {
		bus.Handshake.Subscribe(func(event HandshakeEvent) (Decision, string) {
			result := handler(event)
			return result.Decision, result.Payload
		})
	}
	for _, handler := range r.disconnect {
		bus.Disconnect.Subscribe(func(event DisconnectEvent) (Decision, string) {
			result := handler(event)
			return result.Decision, result.Payload
		})
	}
}
