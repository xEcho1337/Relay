package abi

import (
	"encoding/binary"
	"errors"
)

const responseSize = 8

const loginResponseMarker byte = 0xA1

func EncodeResponse(decision uint8, reason string) ([]byte, error) {
	if decision > 2 {
		return nil, errors.New("invalid plugin decision")
	}

	data := make([]byte, responseSize+len(reason))
	binary.LittleEndian.PutUint16(data[:2], Version)
	data[2] = decision
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(reason)))

	copy(data[responseSize:], reason)
	return data, nil
}

func DecodeResponse(data []byte) (uint8, string, error) {
	if len(data) < responseSize {
		return 0, "", errors.New("plugin response is truncated")
	}

	if binary.LittleEndian.Uint16(data[:2]) != Version {
		return 0, "", errors.New("unsupported plugin ABI version")
	}

	if data[2] > 2 {
		return 0, "", errors.New("invalid plugin decision")
	}

	reasonSize := int(binary.LittleEndian.Uint32(data[4:8]))
	if responseSize+reasonSize > len(data) {
		return 0, "", errors.New("invalid plugin response length")
	}

	return data[2], string(data[responseSize : responseSize+reasonSize]), nil
}

// EncodeLoginResponse includes login fields changed by a plugin.
func EncodeLoginResponse(decision uint8, reason, username, uuid, backend string, changed uint8) ([]byte, error) {
	base, err := EncodeResponse(decision, reason)
	if err != nil {
		return nil, err
	}
	if changed == 0 {
		return base, nil
	}
	data := append(base, loginResponseMarker, changed)
	for _, value := range []string{username, uuid, backend} {
		if len(value) > 65535 {
			return nil, errors.New("login response field too long")
		}
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], uint16(len(value)))
		data = append(data, b[:]...)
		data = append(data, value...)
	}
	return data, nil
}

// DecodeLoginResponse decodes a response and any changed login fields.
func DecodeLoginResponse(data []byte) (uint8, string, string, string, string, uint8, error) {
	decision, reason, err := DecodeResponse(data[:responseSize+int(binary.LittleEndian.Uint32(data[4:8]))])
	if err != nil {
		return 0, "", "", "", "", 0, err
	}
	pos := responseSize + len(reason)
	if len(data) < pos+2 || data[pos] != loginResponseMarker {
		return decision, reason, "", "", "", 0, nil
	}
	changed := data[pos+1]
	pos += 2
	values := make([]string, 3)
	for i := range values {
		if pos+2 > len(data) {
			return 0, "", "", "", "", 0, errors.New("truncated login response")
		}
		n := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
		pos += 2
		if pos+n > len(data) {
			return 0, "", "", "", "", 0, errors.New("truncated login response field")
		}
		values[i] = string(data[pos : pos+n])
		pos += n
	}
	if pos != len(data) {
		return 0, "", "", "", "", 0, errors.New("invalid login response length")
	}
	return decision, reason, values[0], values[1], values[2], changed, nil
}
