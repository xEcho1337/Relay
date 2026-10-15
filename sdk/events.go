package sdk

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
}

type LoginEvent struct {
	ConnectionID uint64
	Username     string
	UUID         string
	Backend      string
}

type DisconnectEvent struct {
	ConnectionID uint64
	Username     string
	UUID         string
	Reason       string
}
type Decision int

const (
	Allow Decision = iota
	OverrideBackend
	Deny
)
