package connection

import (
	"fmt"
	"sync"
	"testing"

	"relay/sdk"
)

func testConnPlayer() sdk.Player {
	return sdk.Player{
		Username:    "Steve",
		UUID:        "uuid-1",
		Backend:     "127.0.0.1:25566",
		ClientIP:    "192.0.2.10",
		ConnectedAt: 1729000000000,
	}
}

func TestManagerLifecycle(t *testing.T) {
	m := NewManager()

	status := m.Add("192.0.2.1:1234")
	if status.State != sdk.StateHandshake {
		t.Fatalf("initial state = %v, want handshake", status.State)
	}
	if status.HasPlayer {
		t.Fatal("fresh connection has a player")
	}
	if status.ID == 0 {
		t.Fatal("connection ID not assigned")
	}

	if !m.SetProtocol(status.ID, 767) {
		t.Fatal("SetProtocol failed")
	}
	if !m.SetState(status.ID, sdk.StateStatus) {
		t.Fatal("SetState failed")
	}
	if !m.SetBackend(status.ID, "127.0.0.1:25566") {
		t.Fatal("SetBackend failed")
	}

	got, ok := m.Get(status.ID)
	if !ok {
		t.Fatal("Get failed")
	}
	if got.Protocol != 767 || got.State != sdk.StateStatus || got.Backend != "127.0.0.1:25566" {
		t.Fatalf("Get = %+v, fields not updated", got)
	}
	if got.HasPlayer {
		t.Fatal("status connection must stay player-less")
	}
	if m.CountInState(sdk.StateStatus) != 1 {
		t.Fatal("CountInState(status) != 1")
	}

	login := m.Add("192.0.2.2:2345")
	m.SetProtocol(login.ID, 767)
	m.SetState(login.ID, sdk.StateLogin)
	if !m.SetPlayer(login.ID, testConnPlayer()) {
		t.Fatal("SetPlayer failed")
	}
	m.SetState(login.ID, sdk.StatePlay)

	if m.Count() != 2 {
		t.Fatalf("Count = %d, want 2", m.Count())
	}
	if m.CountInState(sdk.StatePlay) != 1 {
		t.Fatal("CountInState(play) != 1")
	}

	if !m.Remove(status.ID) || !m.Remove(login.ID) {
		t.Fatal("Remove failed")
	}
	if m.Remove(login.ID) {
		t.Fatal("double Remove reported success")
	}
	if m.Count() != 0 {
		t.Fatal("table not empty after Remove")
	}
}

func TestManagerRejectsBadUpdates(t *testing.T) {
	m := NewManager()
	c := m.Add("192.0.2.3:3456")

	bad := testConnPlayer()
	bad.Username = ""
	if m.SetPlayer(c.ID, bad) {
		t.Fatal("SetPlayer accepted an invalid player")
	}
	if _, ok := m.Get(c.ID); !ok {
		t.Fatal("entry lost after rejected SetPlayer")
	} else if got, _ := m.Get(c.ID); got.HasPlayer {
		t.Fatal("rejected SetPlayer still attached a player")
	}

	if m.SetBackend(c.ID, "") {
		t.Fatal("SetBackend accepted an empty address")
	}

	const ghost = 9999
	if m.SetProtocol(ghost, 47) || m.SetState(ghost, sdk.StatePlay) ||
		m.SetPlayer(ghost, testConnPlayer()) || m.SetBackend(ghost, "x") || m.Remove(ghost) {
		t.Fatal("update on unknown ID reported success")
	}
	if _, ok := m.Get(ghost); ok {
		t.Fatal("Get on unknown ID reported found")
	}
}

func TestManagerListSnapshot(t *testing.T) {
	m := NewManager()
	a := m.Add("192.0.2.1:1")
	m.SetPlayer(a.ID, testConnPlayer())

	list := m.List()
	if len(list) != 1 {
		t.Fatalf("len(List) = %d, want 1", len(list))
	}
	list[0].State = sdk.StateClosed
	list[0].Player.Username = "HACKED"

	got, _ := m.Get(a.ID)
	if got.State == sdk.StateClosed || got.Player.Username == "HACKED" {
		t.Fatal("List: mutation of snapshot leaked into manager")
	}
}

func TestManagerConcurrent(t *testing.T) {
	m := NewManager()
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := m.Add(fmt.Sprintf("192.0.2.%d:1234", i))
			m.SetProtocol(c.ID, 767)
			m.SetState(c.ID, sdk.StateLogin)
			p := testConnPlayer()
			p.UUID = fmt.Sprintf("uuid-%d", i)
			p.Username = fmt.Sprintf("P%d", i)
			m.SetPlayer(c.ID, p)
			m.SetBackend(c.ID, "127.0.0.1:25566")
			m.SetState(c.ID, sdk.StatePlay)
			_, _ = m.Get(c.ID)
			_ = m.List()
			_ = m.Count()
			m.Remove(c.ID)
		}(i)
	}
	wg.Wait()
	if m.Count() != 0 {
		t.Fatalf("Count = %d after paired add/remove, want 0", m.Count())
	}
}
