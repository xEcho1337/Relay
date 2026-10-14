package packets

import (
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
