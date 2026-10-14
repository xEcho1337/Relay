package network_test

import (
	"bytes"
	"encoding/binary"
	"relay/network"
	"relay/network/packets"
	"testing"
)

func writeVarInt(buf *bytes.Buffer, v int) {
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if v == 0 {
			return
		}
	}
}

func writeString(buf *bytes.Buffer, s string) {
	writeVarInt(buf, len(s))
	buf.WriteString(s)
}

func buildHandshake(protocol int, address string, port uint16, next int) []byte {
	var payload bytes.Buffer
	writeVarInt(&payload, 0x00) // packet id
	writeVarInt(&payload, protocol)
	writeString(&payload, address)
	binary.Write(&payload, binary.BigEndian, port)
	writeVarInt(&payload, next)

	var packet bytes.Buffer
	writeVarInt(&packet, payload.Len())
	packet.Write(payload.Bytes())
	return packet.Bytes()
}

func TestReadHandshakeLogin(t *testing.T) {
	raw := buildHandshake(47, "localhost", 25565, 2)

	hs, err := packets.ReadHandshake(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("ReadHandshake: %v", err)
	}
	if hs.Protocol != 47 {
		t.Errorf("protocol = %d, want 47", hs.Protocol)
	}
	if hs.Address != "localhost" {
		t.Errorf("address = %q, want \"localhost\"", hs.Address)
	}
	if hs.Port != 25565 {
		t.Errorf("port = %d, want 25565", hs.Port)
	}
	if hs.NextState != 2 {
		t.Errorf("next = %d, want 2", hs.NextState)
	}
	if !bytes.Equal(hs.Raw, raw) {
		t.Errorf("raw does not match original packet")
	}
}

func TestReadHandshakeStatus(t *testing.T) {
	hs, err := packets.ReadHandshake(bytes.NewReader(buildHandshake(47, "play.example.it", 25565, 1)))
	if err != nil {
		t.Fatalf("ReadHandshake: %v", err)
	}
	if hs.NextState != 1 {
		t.Errorf("next = %d, want 1", hs.NextState)
	}
}

func TestReadHandshakeBungeeAddress(t *testing.T) {
	// BungeeCord format: host\0ip\0uuid.
	addr := "localhost\x00127.0.0.1\x00ebff4c8c-7299-3b8f-8d8c-37d5e5e0ca6c"
	hs, err := packets.ReadHandshake(bytes.NewReader(buildHandshake(47, addr, 25565, 2)))
	if err != nil {
		t.Fatalf("ReadHandshake: %v", err)
	}
	if hs.Address != addr {
		t.Errorf("address = %q, want %q", hs.Address, addr)
	}
}

func TestReadHandshakeNotHandshake(t *testing.T) {
	// First packet with id != 0x00 must fail.
	var payload bytes.Buffer
	writeVarInt(&payload, 0x01)
	var packet bytes.Buffer
	writeVarInt(&packet, payload.Len())
	packet.Write(payload.Bytes())

	if _, err := packets.ReadHandshake(bytes.NewReader(packet.Bytes())); err == nil {
		t.Errorf("want error for packet id != 0x00")
	}
}

func TestReadHandshakeTruncated(t *testing.T) {
	// Truncated packet must fail, not block.
	raw := buildHandshake(47, "localhost", 25565, 2)
	if _, err := packets.ReadHandshake(bytes.NewReader(raw[:len(raw)-3])); err == nil {
		t.Errorf("want error for truncated packet")
	}
}

func TestReadVarIntSizes(t *testing.T) {
	cases := []struct {
		in   []byte
		want int
	}{
		{[]byte{0x00}, 0},
		{[]byte{0x2F}, 47},
		{[]byte{0xAC, 0x02}, 300},
		{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}, 0xFFFFFFFF},
	}
	for _, c := range cases {
		got, err := network.NewBufferFromBytes(c.in).ReadVarInt()
		if err != nil {
			t.Fatalf("ReadVarInt(%x): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ReadVarInt(%x) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestReadVarIntTooLong(t *testing.T) {
	if _, err := network.NewBufferFromBytes([]byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x00}).ReadVarInt(); err == nil {
		t.Errorf("want error for 6-byte varint")
	}
	if _, err := network.NewBufferFromBytes(nil).ReadVarInt(); err == nil {
		t.Errorf("want error for empty input")
	}
}
