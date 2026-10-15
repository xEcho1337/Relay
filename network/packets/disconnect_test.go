package packets

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"relay/network"
	"relay/protocol"
	"relay/sdk"
)

func TestPlayDisconnectID(t *testing.T) {
	cases := []struct {
		protocol int
		want     int
	}{
		{4, 0x40}, {5, 0x40}, {47, 0x40}, {66, 0x40},
		{67, 0x1A}, {110, 0x1A}, {210, 0x1A}, {316, 0x1A}, {317, 0x1A},
		{318, 0x1B}, {331, 0x1B},
		{332, 0x1A}, {340, 0x1A}, {344, 0x1A},
		{345, 0x1B}, {404, 0x1B}, {470, 0x1B},
		{471, 0x1A}, {498, 0x1A}, {549, 0x1A},
		{550, 0x1B}, {578, 0x1B}, {734, 0x1B},
		{735, 0x1A}, {736, 0x1A}, {755, 0x1A}, {756, 0x1A}, {757, 0x1A}, {758, 0x1A},
		{759, 0x17},
		{760, 0x19},
		{761, 0x17},
		{762, 0x1A}, {763, 0x1A},
		{764, 0x1B}, {765, 0x1B},
		{766, 0x1D}, {767, 0x1D}, {768, 0x1D}, {769, 0x1D},
		{770, 0x1C}, {771, 0x1C}, {772, 0x1C},
		{773, 0x20}, {774, 0x20}, {775, 0x20}, {776, 0x20}, {777, 0x20},
	}
	for _, c := range cases {
		id, ok := PlayDisconnectID(c.protocol)
		if !ok {
			t.Errorf("protocol %d: unknown, want %#x", c.protocol, c.want)
			continue
		}
		if id != c.want {
			t.Errorf("protocol %d: id = %#x, want %#x", c.protocol, id, c.want)
		}
	}
}

func TestPlayDisconnectIDUnknown(t *testing.T) {
	for _, proto := range []int{-1, 0, 3, 778, 800, 100000} {
		if id, ok := PlayDisconnectID(proto); ok {
			t.Errorf("protocol %d: id = %#x, want unknown", proto, id)
		}
	}
}

func readPacketID(t *testing.T, buf *bytes.Buffer) int {
	t.Helper()
	b, _, err := network.ReadFrame(buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	id, err := b.ReadVarInt()
	if err != nil {
		t.Fatalf("ReadVarInt: %v", err)
	}
	return id
}

func TestWriteDisconnectPerState(t *testing.T) {
	cases := []struct {
		state    sdk.State
		protocol int
		wantID   int
	}{
		{sdk.StateLogin, 47, 0x00},
		{sdk.StateTransfer, 767, 0x00},
		{sdk.StateConfiguration, 764, 0x01},
		{sdk.StateConfiguration, 765, 0x01},
		{sdk.StateConfiguration, 766, 0x02},
		{sdk.StateConfiguration, 777, 0x02},
		{sdk.StatePlay, 47, 0x40},
		{sdk.StatePlay, 340, 0x1A},
		{sdk.StatePlay, 760, 0x19},
		{sdk.StatePlay, 766, 0x1D},
		{sdk.StatePlay, 770, 0x1C},
		{sdk.StatePlay, 777, 0x20},
	}
	for _, c := range cases {
		var buf bytes.Buffer
		ok, err := WriteDisconnect(&buf, c.state, c.protocol, "kicked")
		if err != nil {
			t.Errorf("%v proto %d: err = %v", c.state, c.protocol, err)
			continue
		}
		if !ok {
			t.Errorf("%v proto %d: ok = false, want true", c.state, c.protocol)
			continue
		}
		raw := buf.String()
		if id := readPacketID(t, &buf); id != c.wantID {
			t.Errorf("%v proto %d: id = %#x, want %#x", c.state, c.protocol, id, c.wantID)
		}
		if !strings.Contains(raw, "kicked") {
			t.Errorf("%v proto %d: reason missing from payload", c.state, c.protocol)
		}
	}
}

func TestWriteDisconnectNoPacket(t *testing.T) {
	for _, s := range []sdk.State{sdk.StateHandshake, sdk.StateStatus, sdk.StateClosed} {
		var buf bytes.Buffer
		ok, err := WriteDisconnect(&buf, s, 767, "bye")
		if err != nil {
			t.Errorf("%v: err = %v, want nil", s, err)
		}
		if ok {
			t.Errorf("%v: ok = true, want false (no packet exists)", s)
		}
		if buf.Len() != 0 {
			t.Errorf("%v: wrote %d bytes, want none", s, buf.Len())
		}
	}
}

func TestWriteDisconnectPlayUnknownProtocol(t *testing.T) {
	var buf bytes.Buffer
	ok, err := WriteDisconnect(&buf, sdk.StatePlay, 9999, "bye")
	if ok {
		t.Fatal("ok = true for unknown protocol")
	}
	var unknown *protocol.UnknownProtocolError
	if !errors.As(err, &unknown) {
		t.Fatalf("err = %v, want UnknownProtocolError", err)
	}
	if buf.Len() != 0 {
		t.Fatal("bytes written for unknown protocol")
	}
}

func TestWriteDisconnectConfigUnknownProtocol(t *testing.T) {
	for _, proto := range []int{763, 778} {
		var buf bytes.Buffer
		ok, err := WriteDisconnect(&buf, sdk.StateConfiguration, proto, "bye")
		if ok {
			t.Fatalf("proto %d: ok = true", proto)
		}
		var unknown *protocol.UnknownProtocolError
		if !errors.As(err, &unknown) {
			t.Fatalf("proto %d: err = %v, want UnknownProtocolError", proto, err)
		}
		if buf.Len() != 0 {
			t.Fatalf("proto %d: bytes written", proto)
		}
	}
}

func TestWriteLoginDisconnectKeepsLegacyShape(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteLoginDisconnect(&buf, "nope"); err != nil {
		t.Fatalf("WriteLoginDisconnect: %v", err)
	}
	if id := readPacketID(t, &buf); id != 0x00 {
		t.Fatalf("id = %#x, want 0x00", id)
	}
}
