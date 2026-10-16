package sdk

import (
	"errors"

	"relay/abi"
)

const ABIVersion = abi.Version

type Plugin struct {
	registry Registry
}

func NewPlugin() *Plugin {
	plugin := &Plugin{}
	activePlugin = plugin
	return plugin
}

func (p *Plugin) OnLogin(handler func(LoginEvent) Result) {
	p.registry.OnLogin(handler)
}

func (p *Plugin) OnHandshake(handler func(HandshakeEvent) Result) {
	p.registry.OnHandshake(handler)
}

func (p *Plugin) OnDisconnect(handler func(DisconnectEvent) Result) {
	p.registry.OnDisconnect(handler)
}

func (p *Plugin) HandleMessage(data []byte) ([]byte, error) {
	kind, err := abi.MessageType(data)
	if err != nil {
		return nil, err
	}

	switch kind {
	case abi.Login:
		return p.handleLogin(data)
	case abi.Handshake:
		return p.handleHandshake(data)
	case abi.Disconnect:
		return p.handleDisconnect(data)
	default:
		return nil, errors.New("unsupported plugin message type")
	}
}

func (p *Plugin) handleLogin(data []byte) ([]byte, error) {
	id, username, uuid, backend, err := abi.DecodeLogin(data)
	if err != nil {
		return nil, err
	}
	return p.dispatchLogin(LoginEvent{id, username, uuid, backend})
}

func (p *Plugin) handleHandshake(data []byte) ([]byte, error) {
	id, host, clientIP, protocol, port, state, err := abi.DecodeHandshake(data)
	if err != nil {
		return nil, err
	}
	event := HandshakeEvent{id, protocol, host, port, HandshakeNextState(state), clientIP}
	return p.dispatchHandshake(event)
}

func (p *Plugin) handleDisconnect(data []byte) ([]byte, error) {
	id, username, uuid, reason, err := abi.DecodeDisconnect(data)
	if err != nil {
		return nil, err
	}
	return p.dispatchDisconnect(DisconnectEvent{id, username, uuid, reason})
}

func (p *Plugin) dispatchLogin(event LoginEvent) ([]byte, error) {
	result := dispatch(p.registry.login, event)
	return abi.EncodeResponse(uint8(result.Decision), result.Payload)
}

func (p *Plugin) dispatchHandshake(event HandshakeEvent) ([]byte, error) {
	result := dispatch(p.registry.handshake, event)
	return abi.EncodeResponse(uint8(result.Decision), result.Payload)
}

func (p *Plugin) dispatchDisconnect(event DisconnectEvent) ([]byte, error) {
	result := dispatch(p.registry.disconnect, event)
	return abi.EncodeResponse(uint8(result.Decision), result.Payload)
}

func dispatch[T any](handlers []func(T) Result, event T) Result {
	result := Result{Decision: Allow}
	for _, handler := range handlers {
		result = handler(event)
		if result.Decision == Deny {
			break
		}
	}

	return result
}
