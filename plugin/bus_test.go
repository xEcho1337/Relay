package plugin

import (
	"sync"
	"testing"

	"relay/sdk"
)

func testHandshakeEvent() sdk.HandshakeEvent {
	return sdk.HandshakeEvent{
		ConnectionID: 7,
		Protocol:     767,
		Host:         "lobby.testing.it",
		Port:         25565,
		NextState:    sdk.Login,
		ClientIP:     "192.0.2.10",
	}
}

func testLoginEvent() sdk.LoginEvent {
	return sdk.LoginEvent{
		ConnectionID: 7,
		Username:     "Steve",
		UUID:         "uuid-1",
		Backend:      "127.0.0.1:25566",
	}
}

func allowHandshake(sdk.HandshakeEvent) (sdk.Decision, string) {
	return sdk.Allow, ""
}

func TestEmitWithoutSubscribers(t *testing.T) {
	b := NewBus()
	if d, p := b.Handshake.Emit(testHandshakeEvent()); d != sdk.Allow || p != "" {
		t.Fatalf("handshake = (%d, %q), want (allow, \"\")", d, p)
	}
	if d, p := b.Login.Emit(testLoginEvent()); d != sdk.Allow || p != "" {
		t.Fatalf("login = (%d, %q), want (allow, \"\")", d, p)
	}
	b.Disconnect.Emit(sdk.DisconnectEvent{})
}

func TestHandshakeAllowAndOverride(t *testing.T) {
	b := NewBus()
	b.Handshake.Subscribe(allowHandshake)
	b.Handshake.Subscribe(func(e sdk.HandshakeEvent) (sdk.Decision, string) {
		if e.ConnectionID != 7 || e.Host != "lobby.testing.it" {
			t.Errorf("event = %+v, fields not forwarded", e)
		}
		return sdk.OverrideBackend, "10.0.0.1:25566"
	})
	b.Handshake.Subscribe(func(sdk.HandshakeEvent) (sdk.Decision, string) {
		return sdk.OverrideBackend, "10.0.0.2:25566"
	})

	d, payload := b.Handshake.Emit(testHandshakeEvent())
	if d != sdk.OverrideBackend {
		t.Fatalf("decision = %d, want override", d)
	}
	if payload != "10.0.0.2:25566" {
		t.Fatalf("payload = %q, want last override", payload)
	}
}

func TestDenyShortCircuits(t *testing.T) {
	b := NewBus()
	calls := 0
	b.Login.Subscribe(func(sdk.LoginEvent) (sdk.Decision, string) {
		calls++
		return sdk.OverrideBackend, "10.0.0.1:25566"
	})
	b.Login.Subscribe(func(sdk.LoginEvent) (sdk.Decision, string) {
		calls++
		return sdk.Deny, "banned"
	})
	b.Login.Subscribe(func(sdk.LoginEvent) (sdk.Decision, string) {
		calls++
		return sdk.Allow, ""
	})

	d, reason := b.Login.Emit(testLoginEvent())
	if d != sdk.Deny || reason != "banned" {
		t.Fatalf("got (%d, %q), want (deny, banned)", d, reason)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2 (short-circuit)", calls)
	}
}

func TestSubscribeNilIgnored(t *testing.T) {
	b := NewBus()
	b.Handshake.Subscribe(nil)
	b.Login.Subscribe(nil)
	b.Disconnect.Subscribe(nil)
	if d, _ := b.Handshake.Emit(testHandshakeEvent()); d != sdk.Allow {
		t.Fatal("nil handler changed the decision")
	}
}

func TestDisconnectNotifiesAll(t *testing.T) {
	b := NewBus()
	var got []string
	var mu sync.Mutex
	for _, name := range []string{"a", "b", "c"} {
		name := name
		b.Disconnect.Subscribe(func(e sdk.DisconnectEvent) {
			if e.Username != "Steve" || e.Reason != "quit" {
				t.Errorf("event = %+v, fields not forwarded", e)
			}
			mu.Lock()
			got = append(got, name)
			mu.Unlock()
		})
	}
	b.Disconnect.Emit(sdk.DisconnectEvent{ConnectionID: 7, Username: "Steve", Reason: "quit"})
	if len(got) != 3 {
		t.Fatalf("notified = %d, want 3", len(got))
	}
}

func TestBusConcurrent(t *testing.T) {
	b := NewBus()
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Handshake.Subscribe(allowHandshake)
			_, _ = b.Handshake.Emit(testHandshakeEvent())
			b.Disconnect.Subscribe(func(sdk.DisconnectEvent) {})
			b.Disconnect.Emit(sdk.DisconnectEvent{})
		}()
	}
	wg.Wait()
}
