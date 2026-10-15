package connection

import (
	"net"
	"testing"

	"relay/network/packets"
	"relay/plugin"
	"relay/protocol"
	"relay/sdk"
)

func BenchmarkManagerFullLifecycle(b *testing.B) {
	m := NewManager()
	p := sdk.Player{Username: "Steve", UUID: "uuid-1", Backend: "127.0.0.1:25566"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := m.Add("192.0.2.10:1234")
		m.SetProtocol(c.ID, 767)
		m.SetState(c.ID, protocol.StatePlay)
		m.SetPlayer(c.ID, p)
		m.SetBackend(c.ID, "127.0.0.1:25566")
		m.Remove(c.ID)
	}
}

func BenchmarkBusEmitEmpty(b *testing.B) {
	bus := plugin.NewBus()
	e := sdk.HandshakeEvent{ConnectionID: 1, Protocol: 767, Host: "lobby", Port: 25565}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bus.Handshake.Emit(e)
	}
}

func BenchmarkBusEmitThreeHandlers(b *testing.B) {
	bus := plugin.NewBus()
	allow := func(sdk.HandshakeEvent) (sdk.Decision, string) { return sdk.Allow, "" }
	bus.Handshake.Subscribe(allow)
	bus.Handshake.Subscribe(allow)
	bus.Handshake.Subscribe(allow)
	e := sdk.HandshakeEvent{ConnectionID: 1, Protocol: 767, Host: "lobby", Port: 25565}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bus.Handshake.Emit(e)
	}
}

func BenchmarkPlayDisconnectID(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = packets.PlayDisconnectID(767)
	}
}

func BenchmarkRelayPipe(b *testing.B) {
	const size = 1 << 20
	payload := make([]byte, size)
	a, c := net.Pipe()
	defer a.Close()
	defer c.Close()
	go func() {
		for {
			buf := make([]byte, 32*1024)
			if _, err := a.Read(buf); err != nil {
				return
			}
		}
	}()
	b.SetBytes(size)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := 0
		for n < size {
			m, err := c.Write(payload[n:])
			if err != nil {
				b.Fatalf("write: %v", err)
			}
			n += m
		}
	}
}
