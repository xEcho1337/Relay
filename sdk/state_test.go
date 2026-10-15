package sdk_test

import (
	"testing"

	"relay/sdk"
)

func TestStateString(t *testing.T) {
	cases := map[sdk.State]string{
		sdk.StateHandshake:     "handshake",
		sdk.StateStatus:        "status",
		sdk.StateLogin:         "login",
		sdk.StateTransfer:      "transfer",
		sdk.StateConfiguration: "configuration",
		sdk.StatePlay:          "play",
		sdk.StateClosed:        "closed",
		sdk.State(99):          "unknown",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("State(%d).String() = %q, want %q", int(s), got, want)
		}
	}
}

func TestStateGracefulDisconnect(t *testing.T) {
	withPacket := []sdk.State{
		sdk.StateLogin, sdk.StateTransfer,
		sdk.StateConfiguration, sdk.StatePlay,
	}
	for _, s := range withPacket {
		if !s.GracefulDisconnect() {
			t.Errorf("%v: GracefulDisconnect = false, want true", s)
		}
	}
	withoutPacket := []sdk.State{
		sdk.StateHandshake, sdk.StateStatus, sdk.StateClosed,
	}
	for _, s := range withoutPacket {
		if s.GracefulDisconnect() {
			t.Errorf("%v: GracefulDisconnect = true, want false", s)
		}
	}
}
