package packets

import (
	"bytes"
	"fmt"
	"io"

	"relay/network"
)

// EncryptionRequest Login clientbound 0x01: server -> client.
type EncryptionRequest struct {
	ServerID    string
	PublicKey   []byte
	VerifyToken []byte
}

func WriteEncryptionRequest(w io.Writer, req *EncryptionRequest) error {
	var payload bytes.Buffer
	network.WriteVarInt(&payload, 0x01)
	network.WriteString(&payload, req.ServerID)
	network.WriteByteArray(&payload, req.PublicKey)
	network.WriteByteArray(&payload, req.VerifyToken)
	return network.WriteFrame(w, payload.Bytes())
}

// EncryptionResponse Login serverbound 0x01: client -> server.
type EncryptionResponse struct {
	SharedSecretEncrypted []byte
	VerifyTokenEncrypted  []byte
	Raw                   []byte
}

func ReadEncryptionResponse(r io.Reader) (*EncryptionResponse, error) {
	b, raw, err := network.ReadFrame(r)
	if err != nil {
		return nil, err
	}

	id, err := b.ReadVarInt()
	if err != nil {
		return nil, fmt.Errorf("packet id: %w", err)
	}
	if id != 0x01 {
		return nil, fmt.Errorf("expected encryption response (id=1), got id=%d", id)
	}

	secret, err := b.ReadByteArray(256)
	if err != nil {
		return nil, fmt.Errorf("shared secret: %w", err)
	}
	token, err := b.ReadByteArray(256)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	return &EncryptionResponse{
		SharedSecretEncrypted: secret,
		VerifyTokenEncrypted:  token,
		Raw:                   raw,
	}, nil
}
