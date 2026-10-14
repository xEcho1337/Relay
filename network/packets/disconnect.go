package packets

import (
	"bytes"
	"encoding/json"
	"io"

	"relay/network"
)

// WriteLoginDisconnect sends login clientbound 0x00 with a plain-text
// reason serialized as {"text": "..."}.
func WriteLoginDisconnect(w io.Writer, reason string) error {
	raw, _ := json.Marshal(map[string]string{"text": reason})

	var payload bytes.Buffer
	network.WriteVarInt(&payload, 0x00)
	network.WriteString(&payload, string(raw))
	return network.WriteFrame(w, payload.Bytes())
}
