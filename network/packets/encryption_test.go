package packets

import (
	"bytes"
	"testing"

	"relay/network"
)

func writeAndSkipHeader(t *testing.T, proto int) *network.Buffer {
	t.Helper()
	req := &EncryptionRequest{
		ServerID:    "",
		PublicKey:   []byte{1, 2, 3},
		VerifyToken: []byte{4, 5, 6, 7},
	}
	var buf bytes.Buffer
	if err := WriteEncryptionRequest(&buf, req, proto); err != nil {
		t.Fatalf("protocol %d: %v", proto, err)
	}
	b, _, err := network.ReadFrame(&buf)
	if err != nil {
		t.Fatalf("protocol %d ReadFrame: %v", proto, err)
	}
	if id, err := b.ReadVarInt(); err != nil || id != 0x01 {
		t.Fatalf("protocol %d: id = %d, err = %v", proto, id, err)
	}
	if _, err := b.ReadString(); err != nil {
		t.Fatalf("protocol %d server id: %v", proto, err)
	}
	if _, err := b.ReadByteArray(256); err != nil {
		t.Fatalf("protocol %d public key: %v", proto, err)
	}
	if _, err := b.ReadByteArray(256); err != nil {
		t.Fatalf("protocol %d verify token: %v", proto, err)
	}
	return b
}

func TestWriteEncryptionRequestAuthenticateFlag(t *testing.T) {
	b := writeAndSkipHeader(t, 765)
	if _, err := b.ReadVarInt(); err == nil {
		t.Fatal("protocol 765: trailing byte present, want none")
	}

	for _, proto := range []int{766, 767, 777} {
		b := writeAndSkipHeader(t, proto)
		flag, err := b.ReadVarInt()
		if err != nil {
			t.Fatalf("protocol %d: missing shouldAuthenticate: %v", proto, err)
		}
		if flag != 1 {
			t.Fatalf("protocol %d: shouldAuthenticate = %d, want 1", proto, flag)
		}
		if _, err := b.ReadVarInt(); err == nil {
			t.Fatalf("protocol %d: extra trailing bytes", proto)
		}
	}
}
