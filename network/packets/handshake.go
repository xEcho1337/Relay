package packets

import (
	"bytes"
	"fmt"
	"io"
	"relay/network"
)

type Handshake struct {
	Protocol  int
	Address   string
	Port      uint16
	NextState int
	Raw       []byte
}

func ReadHandshake(r io.Reader) (*Handshake, error) {
	b, raw, err := network.ReadFrame(r)

	if err != nil {
		return nil, err
	}

	id, err := b.ReadVarInt()

	if err != nil {
		return nil, fmt.Errorf("packet id: %w", err)
	}

	if id != 0x00 {
		return nil, fmt.Errorf("first packet is not handshake (id=%d)", id)
	}

	hs := &Handshake{Raw: raw}

	if hs.Protocol, err = b.ReadVarInt(); err != nil {
		return nil, fmt.Errorf("protocol version: %w", err)
	}

	if hs.Address, err = b.ReadString(); err != nil {
		return nil, fmt.Errorf("server address: %w", err)
	}

	if hs.Port, err = b.ReadUShort(); err != nil {
		return nil, fmt.Errorf("server port: %w", err)
	}

	if hs.NextState, err = b.ReadVarInt(); err != nil {
		return nil, fmt.Errorf("next state: %w", err)
	}

	return hs, nil
}

func WriteHandshake(w io.Writer, hs *Handshake) error {
	var payload bytes.Buffer
	network.WriteVarInt(&payload, 0x00)
	network.WriteVarInt(&payload, hs.Protocol)
	network.WriteString(&payload, hs.Address)
	network.WriteUShort(&payload, hs.Port)
	network.WriteVarInt(&payload, hs.NextState)
	return network.WriteFrame(w, payload.Bytes())
}
