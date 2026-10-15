package sdk_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"relay/sdk"
)

func testPlayer(uuid, username string) sdk.Player {
	return sdk.Player{
		Username:    username,
		UUID:        uuid,
		Backend:     "127.0.0.1:25566",
		ClientIP:    "192.0.2.10",
		ConnectedAt: 1729000000000,
	}
}

func TestAddGet(t *testing.T) {
	s := sdk.NewStore()
	want := testPlayer("uuid-1", "Steve")
	if err := s.Add(want); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, ok := s.Get("uuid-1")
	if !ok {
		t.Fatal("Get: missing just-added player")
	}
	if got != want {
		t.Fatalf("Get = %+v, want %+v", got, want)
	}
	if _, ok := s.Get("nope"); ok {
		t.Fatal("Get: unknown uuid reported found")
	}
}

func TestAddInvalid(t *testing.T) {
	cases := map[string]sdk.Player{
		"empty username":    testPlayer("uuid-1", ""),
		"username too long": testPlayer("uuid-1", strings.Repeat("a", sdk.MaxUsernameLen+1)),
		"empty uuid":        testPlayer("", "Steve"),
	}
	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			s := sdk.NewStore()
			if err := s.Add(p); err == nil {
				t.Fatal("Add: expected error, got nil")
			}
			if s.Count() != 0 {
				t.Fatal("Add: invalid player was stored")
			}
		})
	}
}

func TestAddReplacesSameUUID(t *testing.T) {
	s := sdk.NewStore()
	if err := s.Add(testPlayer("uuid-1", "Steve")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	moved := testPlayer("uuid-1", "Steve")
	moved.Backend = "127.0.0.1:25567"
	if err := s.Add(moved); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _ := s.Get("uuid-1")
	if got.Backend != "127.0.0.1:25567" {
		t.Fatalf("Backend = %q after re-add", got.Backend)
	}
	if s.Count() != 1 {
		t.Fatalf("Count = %d, want 1", s.Count())
	}
}

func TestRemove(t *testing.T) {
	s := sdk.NewStore()
	if s.Remove("ghost") {
		t.Fatal("Remove: unknown uuid reported removed")
	}
	if err := s.Add(testPlayer("uuid-1", "Steve")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !s.Remove("uuid-1") {
		t.Fatal("Remove: existing uuid reported missing")
	}
	if _, ok := s.Get("uuid-1"); ok {
		t.Fatal("Get: removed player still present")
	}
}

func TestListSnapshotSemantics(t *testing.T) {
	s := sdk.NewStore()
	if got := s.List(); got == nil || len(got) != 0 {
		t.Fatalf("empty List = %#v, want empty non-nil slice", got)
	}
	if err := s.Add(testPlayer("uuid-1", "Steve")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(testPlayer("uuid-2", "Alex")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("len(List) = %d, want 2", len(list))
	}
	list[0].Username = "HACKED"
	again := s.List()
	for _, p := range again {
		if p.Username == "HACKED" {
			t.Fatal("List: mutation of snapshot leaked into store")
		}
	}
}

func TestDefaultHelpers(t *testing.T) {
	sdk.Default.Clear()
	defer sdk.Default.Clear()

	if err := sdk.Add(testPlayer("uuid-9", "Steve")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if sdk.Count() != 1 {
		t.Fatalf("Count = %d, want 1", sdk.Count())
	}
	if _, ok := sdk.Get("uuid-9"); !ok {
		t.Fatal("Get: missing player in Default store")
	}
	if len(sdk.List()) != 1 {
		t.Fatal("List: wrong size for Default store")
	}
	if !sdk.Remove("uuid-9") {
		t.Fatal("Remove: failed on Default store")
	}
}

type fakeExecutor struct {
	mu     sync.Mutex
	kicks  map[string]string
	msgs   map[string]string
	allow  bool
	kickOK bool
	msgOK  bool
}

func newFakeExecutor() *fakeExecutor {
	return &fakeExecutor{kicks: map[string]string{}, msgs: map[string]string{}}
}

func (f *fakeExecutor) Kick(uuid, reason string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kicks[uuid] = reason
	return f.kickOK
}

func (f *fakeExecutor) SendMessage(uuid, text string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs[uuid] = text
	return f.msgOK
}

func TestKickWithoutExecutor(t *testing.T) {
	sdk.SetExecutor(nil)
	if sdk.Kick("uuid-1", "stop") {
		t.Fatal("Kick without executor reported true")
	}
	if sdk.Kick("", "stop") {
		t.Fatal("Kick with empty uuid reported true")
	}
	if sdk.Kick("uuid-1", strings.Repeat("x", sdk.MaxReasonLen+1)) {
		t.Fatal("Kick with oversized reason reported true")
	}
}

func TestKickWithExecutor(t *testing.T) {
	f := newFakeExecutor()
	f.kickOK = true
	sdk.SetExecutor(f)
	defer sdk.SetExecutor(nil)

	if !sdk.Kick("uuid-1", "griefing") {
		t.Fatal("Kick reported false")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.kicks["uuid-1"] != "griefing" {
		t.Fatalf("reason = %q, want griefing", f.kicks["uuid-1"])
	}
}

func TestSendMessageNoV1Support(t *testing.T) {
	sdk.SetExecutor(nil)
	if sdk.SendMessage("uuid-1", "hi") {
		t.Fatal("SendMessage without executor reported true")
	}
	f := newFakeExecutor()
	sdk.SetExecutor(f)
	defer sdk.SetExecutor(nil)
	if sdk.SendMessage("uuid-1", "hi") {
		t.Fatal("SendMessage reported true while host refuses")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.msgs["uuid-1"] != "hi" {
		t.Fatal("SendMessage was not forwarded to the executor")
	}
}

func TestConcurrentAddRemove(t *testing.T) {
	s := sdk.NewStore()
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			uuid := fmt.Sprintf("uuid-%d", i)
			_ = s.Add(testPlayer(uuid, fmt.Sprintf("P%d", i)))
			_, _ = s.Get(uuid)
			_ = s.List()
			_ = s.Count()
			_ = s.Remove(uuid)
		}(i)
	}
	wg.Wait()
	if s.Count() != 0 {
		t.Fatalf("Count = %d after paired add/remove, want 0", s.Count())
	}
}
