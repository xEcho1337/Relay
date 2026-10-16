package sdk

import (
	"relay/abi"
	"testing"
)

func TestRegistryRegistersLoginHandler(t *testing.T) {
	registry := NewRegistry()
	registry.OnLogin(func(event LoginEvent) Result {
		if event.Username == "Pippo" {
			return Result{Decision: Deny, Payload: "Username not valid"}
		}
		return Result{Decision: Allow}
	})

	bus := NewBus()
	registry.Register(bus)
	decision, payload := bus.Login.Emit(LoginEvent{Username: "Pippo"})
	if decision != Deny || payload != "Username not valid" {
		t.Fatalf("got (%d, %q), want deny and reason", decision, payload)
	}
}

func TestPluginHandleMessage(t *testing.T) {
	plugin := NewPlugin()
	plugin.OnLogin(func(event LoginEvent) Result {
		if event.Username == "Pippo" {
			return Result{Decision: Deny, Payload: "Username not valid"}
		}
		return Result{Decision: Allow}
	})

	request, err := abi.EncodeLogin(0, "Pippo", "", "")
	if err != nil {
		t.Fatal(err)
	}
	response, err := plugin.HandleMessage(request)
	if err != nil {
		t.Fatal(err)
	}
	decision, payload, err := abi.DecodeResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if Decision(decision) != Deny || payload != "Username not valid" {
		t.Fatalf("got (%d, %q), want deny and reason", decision, payload)
	}
}

func TestPluginHandleMessageTypes(t *testing.T) {
	p := NewPlugin()
	p.OnHandshake(func(event HandshakeEvent) Result {
		if event.Host == "blocked.example" {
			return Result{Decision: Deny, Payload: "blocked host"}
		}
		return Result{Decision: Allow}
	})
	p.OnDisconnect(func(event DisconnectEvent) Result {
		if event.Reason == "bad" {
			return Result{Decision: Deny, Payload: "bad reason"}
		}
		return Result{Decision: Allow}
	})

	handshake, err := abi.EncodeHandshake(0, "blocked.example", "", 0, 0, uint16(Login))
	if err != nil {
		t.Fatal(err)
	}
	disconnect, err := abi.EncodeDisconnect(0, "", "", "bad")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		request []byte
		want    string
	}{
		{"handshake", handshake, "blocked host"},
		{"disconnect", disconnect, "bad reason"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := p.HandleMessage(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			decision, payload, err := abi.DecodeResponse(response)
			if err != nil {
				t.Fatal(err)
			}
			if Decision(decision) != Deny || payload != tc.want {
				t.Fatalf("got (%d, %q), want deny and %q", decision, payload, tc.want)
			}
		})
	}
}
