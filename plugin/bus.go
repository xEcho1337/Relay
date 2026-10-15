package plugin

import (
	"sync"

	"relay/sdk"
)

type HandshakeHandler func(e sdk.HandshakeEvent) (sdk.Decision, string)
type LoginHandler func(e sdk.LoginEvent) (sdk.Decision, string)
type DisconnectHandler func(e sdk.DisconnectEvent)

type HandshakeTopic struct {
	mu       sync.RWMutex
	handlers []HandshakeHandler
}

func (t *HandshakeTopic) Subscribe(h HandshakeHandler) {
	if h == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers = append(t.handlers, h)
}

func (t *HandshakeTopic) Emit(e sdk.HandshakeEvent) (sdk.Decision, string) {
	t.mu.RLock()
	handlers := make([]HandshakeHandler, len(t.handlers))
	copy(handlers, t.handlers)
	t.mu.RUnlock()

	decision := sdk.Allow
	payload := ""
	for _, h := range handlers {
		switch d, p := h(e); d {
		case sdk.Deny:
			return sdk.Deny, p
		case sdk.OverrideBackend:
			decision = sdk.OverrideBackend
			payload = p
		}
	}
	return decision, payload
}

type LoginTopic struct {
	mu       sync.RWMutex
	handlers []LoginHandler
}

func (t *LoginTopic) Subscribe(h LoginHandler) {
	if h == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers = append(t.handlers, h)
}

func (t *LoginTopic) Emit(e sdk.LoginEvent) (sdk.Decision, string) {
	t.mu.RLock()
	handlers := make([]LoginHandler, len(t.handlers))
	copy(handlers, t.handlers)
	t.mu.RUnlock()

	decision := sdk.Allow
	payload := ""
	for _, h := range handlers {
		switch d, p := h(e); d {
		case sdk.Deny:
			return sdk.Deny, p
		case sdk.OverrideBackend:
			decision = sdk.OverrideBackend
			payload = p
		}
	}
	return decision, payload
}

type DisconnectTopic struct {
	mu       sync.RWMutex
	handlers []DisconnectHandler
}

func (t *DisconnectTopic) Subscribe(h DisconnectHandler) {
	if h == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers = append(t.handlers, h)
}

func (t *DisconnectTopic) Emit(e sdk.DisconnectEvent) {
	t.mu.RLock()
	handlers := make([]DisconnectHandler, len(t.handlers))
	copy(handlers, t.handlers)
	t.mu.RUnlock()

	for _, h := range handlers {
		h(e)
	}
}

type Bus struct {
	Handshake  HandshakeTopic
	Login      LoginTopic
	Disconnect DisconnectTopic
}

func NewBus() *Bus {
	return &Bus{}
}

var DefaultBus = NewBus()
