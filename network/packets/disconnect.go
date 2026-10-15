package packets

import (
	"bytes"
	"encoding/json"
	"io"
	"relay/protocol"

	"relay/network"
	"relay/sdk"
)

func configDisconnectID(proto int) (int, bool) {
	if proto < 764 || proto > protocol.MaxKnownProtocol {
		return 0, false
	}
	if proto < 766 {
		return 0x01, true
	}
	return 0x02, true
}

func writeChatDisconnect(w io.Writer, id int, reason string) error {
	raw, _ := json.Marshal(map[string]string{"text": reason})

	var payload bytes.Buffer
	network.WriteVarInt(&payload, id)
	network.WriteString(&payload, string(raw))
	return network.WriteFrame(w, payload.Bytes())
}

func PlayDisconnectID(proto int) (id int, ok bool) {
	if proto < protocol.PlayDisconnectTable[0].MinProtocol || proto > protocol.MaxKnownProtocol {
		return 0, false
	}
	id = protocol.PlayDisconnectTable[0].PacketID

	for _, e := range protocol.PlayDisconnectTable[1:] {
		if proto < e.MinProtocol {
			break
		}
		id = e.PacketID
	}

	return id, true
}

func WriteLoginDisconnect(w io.Writer, reason string) error {
	return writeChatDisconnect(w, protocol.LoginDisconnectId, reason)
}

func WriteLoginDisconnectJSON(w io.Writer, raw string) error {
	var payload bytes.Buffer
	network.WriteVarInt(&payload, protocol.LoginDisconnectId)
	network.WriteString(&payload, raw)
	return network.WriteFrame(w, payload.Bytes())
}

func WriteConfigDisconnect(w io.Writer, proto int, reason string) error {
	id, ok := configDisconnectID(proto)
	if !ok {
		return &protocol.UnknownProtocolError{}
	}
	return writeChatDisconnect(w, id, reason)
}

func WritePlayDisconnect(w io.Writer, proto int, reason string) error {
	id, ok := PlayDisconnectID(proto)
	if !ok {
		return &protocol.UnknownProtocolError{}
	}
	return writeChatDisconnect(w, id, reason)
}

func WriteDisconnect(w io.Writer, state sdk.State, proto int, reason string) (bool, error) {
	switch state {
	case sdk.StateLogin, sdk.StateTransfer:
		return true, WriteLoginDisconnect(w, reason)
	case sdk.StateConfiguration:
		if err := WriteConfigDisconnect(w, proto, reason); err != nil {
			return false, err
		}
		return true, nil
	case sdk.StatePlay:
		if err := WritePlayDisconnect(w, proto, reason); err != nil {
			return false, err
		}
		return true, nil
	default:
		return false, nil
	}
}
