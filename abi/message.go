package abi

import (
	"encoding/binary"
	"errors"
)

const Version uint16 = 1

const (
	Login uint16 = iota + 1
	Handshake
	Disconnect
)

const headerSize = 16

func EncodeLogin(id uint64, username, uuid, backend string) ([]byte, error) {
	return encodeMessage(Login, id, username, uuid, backend)
}

func EncodeDisconnect(id uint64, username, uuid, reason string) ([]byte, error) {
	return encodeMessage(Disconnect, id, username, uuid, reason)
}

func EncodeHandshake(id uint64, host, clientIP string, protocol uint32, port, state uint16) ([]byte, error) {
	if state == 0 {
		return nil, errors.New("invalid handshake state")
	}
	return encodeMessage(Handshake, id, host, clientIP, encodeUint32(protocol), encodeUint16(port), encodeUint16(state))
}

func encodeMessage(kind uint16, id uint64, fields ...string) ([]byte, error) {
	payloadSize := 0
	for _, field := range fields {
		if len(field) > 65535 {
			return nil, errors.New("field exceeds 65535 bytes")
		}
		payloadSize += 2 + len(field)
	}

	message := make([]byte, headerSize+payloadSize)
	binary.LittleEndian.PutUint16(message[0:2], Version)
	binary.LittleEndian.PutUint16(message[2:4], kind)
	binary.LittleEndian.PutUint64(message[4:12], id)
	binary.LittleEndian.PutUint32(message[12:16], uint32(payloadSize))

	position := headerSize
	for _, field := range fields {
		binary.LittleEndian.PutUint16(message[position:position+2], uint16(len(field)))
		position += 2
		copy(message[position:], field)
		position += len(field)
	}
	return message, nil
}

func MessageType(message []byte) (uint16, error) {
	if len(message) < headerSize {
		return 0, errors.New("plugin message is truncated")
	}
	if binary.LittleEndian.Uint16(message[:2]) != Version {
		return 0, errors.New("unsupported plugin ABI version")
	}
	if int(binary.LittleEndian.Uint32(message[12:16])) != len(message)-headerSize {
		return 0, errors.New("invalid plugin message length")
	}
	return binary.LittleEndian.Uint16(message[2:4]), nil
}

func decodeFields(message []byte, kind uint16, count int) ([]string, uint64, error) {
	messageKind, err := MessageType(message)
	if err != nil || messageKind != kind {
		return nil, 0, errors.New("unsupported plugin message")
	}

	fields := make([]string, 0, count)
	position := headerSize
	for range count {
		if position+2 > len(message) {
			return nil, 0, errors.New("truncated plugin field")
		}
		length := int(binary.LittleEndian.Uint16(message[position : position+2]))
		position += 2
		if position+length > len(message) {
			return nil, 0, errors.New("truncated plugin field value")
		}
		fields = append(fields, string(message[position:position+length]))
		position += length
	}
	if position != len(message) {
		return nil, 0, errors.New("invalid plugin payload")
	}
	return fields, binary.LittleEndian.Uint64(message[4:12]), nil
}

func encodeUint32(value uint32) string {
	var encoded [4]byte
	binary.LittleEndian.PutUint32(encoded[:], value)
	return string(encoded[:])
}

func encodeUint16(value uint16) string {
	var encoded [2]byte
	binary.LittleEndian.PutUint16(encoded[:], value)
	return string(encoded[:])
}
