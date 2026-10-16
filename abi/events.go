package abi

import (
	"encoding/binary"
	"errors"
)

func DecodeLogin(message []byte) (uint64, string, string, string, error) {
	fields, id, err := decodeFields(message, Login, 3)
	if err != nil {
		return 0, "", "", "", err
	}
	return id, fields[0], fields[1], fields[2], nil
}

func DecodeDisconnect(message []byte) (uint64, string, string, string, error) {
	fields, id, err := decodeFields(message, Disconnect, 3)
	if err != nil {
		return 0, "", "", "", err
	}
	return id, fields[0], fields[1], fields[2], nil
}

func DecodeHandshake(message []byte) (uint64, string, string, uint32, uint16, uint16, error) {
	fields, id, err := decodeFields(message, Handshake, 5)
	if err != nil {
		return 0, "", "", 0, 0, 0, err
	}
	if len(fields[2]) != 4 || len(fields[3]) != 2 || len(fields[4]) != 2 {
		return 0, "", "", 0, 0, 0, errors.New("invalid handshake payload")
	}
	return id,
		fields[0],
		fields[1],
		binary.LittleEndian.Uint32([]byte(fields[2])),
		binary.LittleEndian.Uint16([]byte(fields[3])),
		binary.LittleEndian.Uint16([]byte(fields[4])),
		nil
}
