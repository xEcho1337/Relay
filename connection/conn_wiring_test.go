package connection

import (
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"relay/config"
	"relay/network"
	"relay/network/packets"
	"relay/plugin"
	"relay/protocol"
	"relay/sdk"
)

func wiringConfig() *config.Config {
	return &config.Config{
		Network: config.NetworkConfig{
			Port:         25565,
			IPForwarding: false,
			OnlineMode:   false,
		},
		Servers: map[string]config.ServerConfig{
			"lobby": {ID: "lobby", Address: "127.0.0.1:25566"},
		},
		Messages: config.MessagesConfig{
			NoServersConfigured:        "none",
			ServerOffline:              "offline",
			UsernameVerificationFailed: "nope",
		},
	}
}

func loopback(t *testing.T) (client, server net.Conn, cleanup func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		accepted <- c
	}()
	client, err = net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	select {
	case server = <-accepted:
	case <-time.After(5 * time.Second):
		t.Fatal("accept timed out")
	}
	return client, server, func() {
		client.Close()
		server.Close()
		ln.Close()
	}
}

func writeLoginStart(t *testing.T, w io.Writer, username string) {
	t.Helper()
	var payload bytes.Buffer
	network.WriteVarInt(&payload, 0x00)
	network.WriteString(&payload, username)
	if err := network.WriteFrame(w, payload.Bytes()); err != nil {
		t.Fatalf("write login start: %v", err)
	}
}

func readPacket(t *testing.T, r io.Reader) (int, []byte) {
	t.Helper()
	b, raw, err := network.ReadFrame(r)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	id, err := b.ReadVarInt()
	if err != nil {
		t.Fatalf("read id: %v", err)
	}
	return id, raw
}

func serveOnce(bus *plugin.Bus, srv net.Conn, cfg *config.Config) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		HandleConnection(srv, cfg, bus)
	}()
	return done
}

func TestHandshakeEmitStatus(t *testing.T) {
	bus := plugin.NewBus()
	var seen []sdk.HandshakeEvent
	bus.Handshake.Subscribe(func(e sdk.HandshakeEvent) (sdk.Decision, string) {
		seen = append(seen, e)
		return sdk.Allow, ""
	})
	var disconnects []sdk.DisconnectEvent
	bus.Disconnect.Subscribe(func(e sdk.DisconnectEvent) {
		disconnects = append(disconnects, e)
	})

	cfg := wiringConfig()
	cfg.Servers = map[string]config.ServerConfig{}

	client, srv, cleanup := loopback(t)
	defer cleanup()
	done := serveOnce(bus, srv, cfg)

	if err := packets.WriteHandshake(client, &packets.Handshake{
		Protocol: 767, Address: "lobby", Port: 25565, NextState: 1,
	}); err != nil {
		t.Fatalf("write handshake: %v", err)
	}
	_, _ = readPacket(t, client)
	<-done

	if len(seen) != 1 {
		t.Fatalf("handshake events = %d, want 1", len(seen))
	}
	e := seen[0]
	if e.Host != "lobby" || e.Protocol != 767 || e.Port != 25565 || e.NextState != sdk.Status {
		t.Fatalf("handshake event = %+v, fields wrong", e)
	}
	if e.ClientIP != "127.0.0.1" || e.ConnectionID == 0 {
		t.Fatalf("handshake event = %+v, peer fields wrong", e)
	}
	if len(disconnects) != 0 {
		t.Fatalf("disconnect events = %d, want none (no username known)", len(disconnects))
	}
	if n := DefaultManager.Count(); n != 0 {
		t.Fatalf("tracked connections = %d, want 0", n)
	}
}

func TestLoginDeny(t *testing.T) {
	bus := plugin.NewBus()
	bus.Handshake.Subscribe(func(sdk.HandshakeEvent) (sdk.Decision, string) {
		return sdk.Allow, ""
	})
	bus.Login.Subscribe(func(e sdk.LoginEvent) (sdk.Decision, string) {
		if e.Username != "Steve" || e.Backend != "127.0.0.1:25566" {
			t.Errorf("login event = %+v, fields wrong", e)
		}
		return sdk.Deny, "go away"
	})
	var disconnects []sdk.DisconnectEvent
	bus.Disconnect.Subscribe(func(e sdk.DisconnectEvent) {
		disconnects = append(disconnects, e)
	})

	client, srv, cleanup := loopback(t)
	defer cleanup()
	done := serveOnce(bus, srv, wiringConfig())

	if err := packets.WriteHandshake(client, &packets.Handshake{
		Protocol: 767, Address: "lobby", Port: 25565, NextState: 2,
	}); err != nil {
		t.Fatalf("write handshake: %v", err)
	}
	writeLoginStart(t, client, "Steve")

	id, raw := readPacket(t, client)
	if id != 0x00 {
		t.Fatalf("kick id = %#x, want login disconnect 0x00", id)
	}
	if !strings.Contains(string(raw), "go away") {
		t.Fatalf("kick payload missing reason: %q", raw)
	}
	<-done

	if len(disconnects) != 1 {
		t.Fatalf("disconnect events = %d, want 1", len(disconnects))
	}
	d := disconnects[0]
	if d.Username != "Steve" || d.Reason != "go away" || d.UUID != offlineUUID("Steve") {
		t.Fatalf("disconnect event = %+v, fields wrong", d)
	}
	if n := DefaultManager.Count(); n != 0 {
		t.Fatalf("tracked connections = %d, want 0", n)
	}
}

func TestLoginOverrideBackend(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("backend listen: %v", err)
	}
	defer backend.Close()
	go func() {
		for {
			c, err := backend.Accept()
			if err != nil {
				return
			}
			go func() {
				_, _ = io.Copy(io.Discard, c)
				c.Close()
			}()
		}
	}()

	bus := plugin.NewBus()
	bus.Handshake.Subscribe(func(sdk.HandshakeEvent) (sdk.Decision, string) {
		return sdk.Allow, ""
	})
	bus.Login.Subscribe(func(sdk.LoginEvent) (sdk.Decision, string) {
		return sdk.OverrideBackend, backend.Addr().String()
	})

	client, srv, cleanup := loopback(t)
	defer cleanup()
	done := serveOnce(bus, srv, wiringConfig())

	if err := packets.WriteHandshake(client, &packets.Handshake{
		Protocol: 767, Address: "lobby", Port: 25565, NextState: 2,
	}); err != nil {
		t.Fatalf("write handshake: %v", err)
	}
	writeLoginStart(t, client, "Alex")

	deadline := time.Now().Add(3 * time.Second)
	for {
		relayed := false
		for _, c := range DefaultManager.List() {
			if c.Backend == backend.Addr().String() && c.State == protocol.StatePlay && c.HasPlayer {
				relayed = true
			}
		}
		if relayed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("override backend never recorded in play state")
		}
		time.Sleep(20 * time.Millisecond)
	}

	client.Close()
	<-done
	if n := DefaultManager.Count(); n != 0 {
		t.Fatalf("tracked connections = %d, want 0", n)
	}
}
