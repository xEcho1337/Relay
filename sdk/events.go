package sdk

type HandshakeNextState int

const (
	Status HandshakeNextState = 1
	Login  HandshakeNextState = 2
)

type HandshakeEvent struct {
	Protocol  uint32
	Host      string
	Port      uint16
	NextState HandshakeNextState // 1=status, 2=login
	ClientIP  string
}

type LoginEvent struct {
	Username string
	UUID     string
	Backend  string
}

type DisconnectEvent struct {
	Username string
	UUID     string
	Reason   string
}
type Decision int

const (
	Allow Decision = iota
	OverrideBackend
	Deny
)

/*
func OnHandshake(func(e HandshakeEvent) Decision)
func OnLogin(func(e LoginEvent) (Decision, string)) // string = backend override o kick reason
func OnDisconnect(func(e DisconnectEvent))
*/
