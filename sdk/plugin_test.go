package sdk

import (
	"testing"

	"github.com/xEcho1337/fable/component"
	"relay/abi"
)

func TestPluginDeny(t *testing.T) {
	plugin := NewPlugin()
	plugin.OnLogin(func(event *LoginEvent) {
		if event.Username == "Pippo" {
			event.Deny(component.Text("Username not valid"))
		}
	})
	request, err := abi.EncodeLogin(0, "Pippo", "", "")
	if err != nil {
		t.Fatal(err)
	}
	response, err := plugin.HandleMessage(request)
	if err != nil {
		t.Fatal(err)
	}
	decision, reason, err := abi.DecodeResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if Decision(decision) != Deny || reason != `{"text":"Username not valid"}` {
		t.Fatalf("got (%d, %q)", decision, reason)
	}
}

func TestPluginCanModifyLogin(t *testing.T) {
	plugin := NewPlugin()
	plugin.OnLogin(func(event *LoginEvent) { event.SetUsername("Steve") })
	request, err := abi.EncodeLogin(0, "Pippo", "", "")
	if err != nil {
		t.Fatal(err)
	}
	response, err := plugin.HandleMessage(request)
	if err != nil {
		t.Fatal(err)
	}
	decision, _, username, _, _, changed, err := abi.DecodeLoginResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if Decision(decision) != Allow {
		t.Fatalf("decision = %d, want Allow", decision)
	}
	if username != "Steve" {
		t.Fatalf("username = %q, want %q", username, "Steve")
	}
	if LoginField(changed)&LoginUsername == 0 {
		t.Fatalf("changed = %d, want LoginUsername bit set", changed)
	}
}
