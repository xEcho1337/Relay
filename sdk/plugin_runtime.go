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

func (p *Plugin) OnLogin(handler func(*LoginEvent)) {
	p.registry.OnLogin(handler)
}

func (p *Plugin) OnHandshake(handler func(*HandshakeEvent)) {
	p.registry.OnHandshake(handler)
}

func (p *Plugin) OnDisconnect(handler func(*DisconnectEvent)) {
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
	event := LoginEvent{ConnectionID: id, Username: username, UUID: uuid, Backend: backend}
	return p.dispatchLogin(&event)
}

func (p *Plugin) handleHandshake(data []byte) ([]byte, error) {
	id, host, clientIP, protocol, port, state, err := abi.DecodeHandshake(data)
	if err != nil {
		return nil, err
	}
	event := HandshakeEvent{ConnectionID: id, Protocol: protocol, Host: host, Port: port, NextState: HandshakeNextState(state), ClientIP: clientIP}
	return p.dispatchHandshake(&event)
}

func (p *Plugin) handleDisconnect(data []byte) ([]byte, error) {
	id, username, uuid, reason, err := abi.DecodeDisconnect(data)
	if err != nil {
		return nil, err
	}
	event := DisconnectEvent{ConnectionID: id, Username: username, UUID: uuid, Reason: reason}
	return p.dispatchDisconnect(&event)
}

func (p *Plugin) dispatchLogin(event *LoginEvent) ([]byte, error) {
	for _, handler := range p.registry.login {
		handler(event)
		if event.decision == Deny {
			break
		}
	}
	return abi.EncodeLoginResponse(uint8(event.decision), reasonString(event.reason), event.Username, event.UUID, event.Backend, uint8(event.changed))
}

func (p *Plugin) dispatchHandshake(event *HandshakeEvent) ([]byte, error) {
	for _, handler := range p.registry.handshake {
		handler(event)
		if event.decision == Deny {
			break
		}
	}
	return abi.EncodeResponse(uint8(event.decision), reasonString(event.reason))
}

func (p *Plugin) dispatchDisconnect(event *DisconnectEvent) ([]byte, error) {
	for _, handler := range p.registry.disconnect {
		handler(event)
		if event.decision == Deny {
			break
		}
	}
	return abi.EncodeResponse(uint8(event.decision), reasonString(event.component))
}
