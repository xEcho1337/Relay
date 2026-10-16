package sdk

import "github.com/xEcho1337/fable/component"

type HandshakeNextState int

const (
	Status   HandshakeNextState = 1
	Login    HandshakeNextState = 2
	Transfer HandshakeNextState = 3
)

type HandshakeEvent struct {
	ConnectionID uint64
	Protocol     uint32
	Host         string
	Port         uint16
	NextState    HandshakeNextState
	ClientIP     string
	decision     Decision
	reason       component.Component
}

type LoginEvent struct {
	ConnectionID uint64
	Username     string
	UUID         string
	Backend      string
	decision     Decision
	reason       component.Component
	changed      LoginField
}

type LoginField uint8

const (
	LoginUsername LoginField = 1 << iota
	LoginUUID
	LoginBackend
)

func (e *LoginEvent) SetUsername(value string) {
	if e.Username != value {
		e.Username = value
		e.changed |= LoginUsername
	}
}

func (e *LoginEvent) SetUUID(value string) {
	if e.UUID != value {
		e.UUID = value
		e.changed |= LoginUUID
	}
}

func (e *LoginEvent) SetBackend(value string) {
	if e.Backend != value {
		e.Backend = value
		e.changed |= LoginBackend
	}
}

func (e *LoginEvent) ChangedFields() LoginField { return e.changed }

type DisconnectEvent struct {
	ConnectionID uint64
	Username     string
	UUID         string
	Reason       string
	decision     Decision
	component    component.Component
}

func (e *LoginEvent) Deny(reason component.Component) {
	e.decision = Deny
	e.reason = reason
}

func (e *LoginEvent) OverrideBackend(backend string) {
	e.decision = OverrideBackend
	e.Backend = backend
}

func (e *LoginEvent) Outcome() (Decision, component.Component) {
	return e.decision, e.reason
}

func (e *HandshakeEvent) Deny(reason component.Component) {
	e.decision = Deny
	e.reason = reason
}

func (e *HandshakeEvent) SetBackend(backend string) {
	e.decision = OverrideBackend
	e.reason = component.Text(backend)
}

func (e *HandshakeEvent) Outcome() (Decision, component.Component) {
	return e.decision, e.reason
}

func (e *DisconnectEvent) Deny(reason component.Component) {
	e.decision = Deny
	e.component = reason
}

func (e *DisconnectEvent) Outcome() (Decision, component.Component) {
	return e.decision, e.component
}

type Decision int

const (
	Allow Decision = iota
	OverrideBackend
	Deny
)
