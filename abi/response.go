package abi

import (
	"encoding/binary"
	"errors"
)

const responseSize = 8

func EncodeResponse(decision uint8, payload string) ([]byte, error) {
	if decision > 2 {
		return nil, errors.New("invalid plugin decision")
	}

	data := make([]byte, responseSize+len(payload))
	binary.LittleEndian.PutUint16(data[:2], Version)
	data[2] = decision
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(payload)))

	copy(data[responseSize:], payload)
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

	payloadSize := int(binary.LittleEndian.Uint32(data[4:8]))
	if responseSize+payloadSize != len(data) {
		return 0, "", errors.New("invalid plugin response length")
	}

	return data[2], string(data[responseSize:]), nil
}
