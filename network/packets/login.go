package packets

import (
	"bytes"
	"fmt"
	"io"

	"relay/network"
)

type LoginStart struct {
	Username string
	Raw      []byte
}

func ReadLoginStart(r io.Reader) (*LoginStart, error) {
	b, raw, err := network.ReadFrame(r)
	if err != nil {
		return nil, err
	}

	id, err := b.ReadVarInt()
	if err != nil {
		return nil, fmt.Errorf("packet id: %w", err)
	}

	if id != 0x00 {
		return nil, fmt.Errorf("first login packet is not login start (id=%d)", id)
	}

	name, err := b.ReadString()
	if err != nil {
		return nil, fmt.Errorf("username: %w", err)
	}

	if name == "" || len(name) > 16 {
		return nil, fmt.Errorf("invalid username %q", name)
	}

	return &LoginStart{Username: name, Raw: raw}, nil
}

// EncodeLoginStart builds a login start packet for the supplied username.
func EncodeLoginStart(username string) []byte {
	var payload bytes.Buffer
	network.WriteVarInt(&payload, 0)
	network.WriteString(&payload, username)
	var packet bytes.Buffer
	network.WriteFrame(&packet, payload.Bytes())
	return packet.Bytes()
}
