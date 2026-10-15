package protocol

const LoginDisconnectId = 0x00
const MaxKnownProtocol = 777

type PacketRegistry struct {
	MinProtocol int
	PacketID    int
}

type UnknownProtocolError struct {
	Protocol int
}

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

var PlayDisconnectTable = []PacketRegistry{
	{4, 0x40},
	{67, 0x1A},
	{318, 0x1B},
	{332, 0x1A},
	{345, 0x1B},
	{471, 0x1A},
	{550, 0x1B},
	{735, 0x1A},
	{759, 0x17},
	{760, 0x19},
	{761, 0x17},
	{762, 0x1A},
	{764, 0x1B},
	{766, 0x1D},
	{770, 0x1C},
	{773, 0x20},
}

func (e *UnknownProtocolError) Error() string {
	return "no known play disconnect packet"
}

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
