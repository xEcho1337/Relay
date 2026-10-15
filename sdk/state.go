package sdk

type State int

const (
	StateHandshake State = iota
	StateStatus
	StateLogin
	// StateTransfer Handshake intent 3 (1.20.5+).
	StateTransfer
	StateConfiguration
	StatePlay
	StateClosed
)

func (s State) String() string {
	switch s {
	case StateHandshake:
		return "handshake"
	case StateStatus:
		return "status"
	case StateLogin:
		return "login"
	case StateTransfer:
		return "transfer"
	case StateConfiguration:
		return "configuration"
	case StatePlay:
		return "play"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

func (s State) GracefulDisconnect() bool {
	switch s {
	case StateLogin, StateTransfer, StateConfiguration, StatePlay:
		return true
	default:
		return false
	}
}
