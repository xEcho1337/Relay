package auth

import (
	"strings"
	"testing"
)

func TestServerHashFormat(t *testing.T) {
	h1 := ServerHash("", bytes16(1), bytes16(2))
	h2 := ServerHash("", bytes16(1), bytes16(2))
	if h1 != h2 {
		t.Fatal("hash not deterministic")
	}
	if h1 == "" || h1 == "-" {
		t.Fatalf("bad hash %q", h1)
	}
	// hex chars + optional leading "-"
	s := strings.TrimPrefix(h1, "-")
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Fatalf("non-hex hash %q", h1)
		}
	}
	if len(s) > 1 && s[0] == '0' {
		t.Fatalf("leading zero in hash %q", h1)
	}
}

func TestDashUUID(t *testing.T) {
	got := DashUUID("ebff4c8c72993b8f8d8c37d5e5e0ca6c")
	want := "ebff4c8c-7299-3b8f-8d8c-37d5e5e0ca6c"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func bytes16(b byte) []byte {
	out := make([]byte, 16)
	for i := range out {
		out[i] = b
	}
	return out
}
