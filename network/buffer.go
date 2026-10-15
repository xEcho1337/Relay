package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Buffer struct {
	r *bytes.Reader
}

func NewBufferFromBytes(b []byte) *Buffer {
	return &Buffer{r: bytes.NewReader(b)}
}

const MaxPacketLength = 1 << 20

func readFrameLength(r io.Reader) (int, []byte, error) {
	var prefix [5]byte

	value := 0
	for i := range len(prefix) {
		if _, err := io.ReadFull(r, prefix[i:i+1]); err != nil {
			return 0, nil, err
		}
		value |= int(prefix[i]&0x7F) << (7 * i)
		if prefix[i]&0x80 == 0 {
			return value, prefix[:i+1], nil
		}
	}

	return 0, nil, fmt.Errorf("varint too long")
}

// ReadFrame reads one length-prefixed frame from the stream,
// returning it as a Buffer plus the raw length-prefixed bytes.
func ReadFrame(r io.Reader) (*Buffer, []byte, error) {
	length, prefix, err := readFrameLength(r)
	if err != nil {
		return nil, nil, err
	}
	if length < 0 || length > MaxPacketLength {
		return nil, nil, fmt.Errorf("invalid packet length: %d", length)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, nil, err
	}

	raw := make([]byte, 0, len(prefix)+length)
	raw = append(raw, prefix...)
	raw = append(raw, payload...)
	return NewBufferFromBytes(payload), raw, nil
}

func (b *Buffer) ReadVarInt() (int, error) {
	value := 0
	for i := range 5 {
		c, err := b.r.ReadByte()

		if err != nil {
			return 0, err
		}

		value |= int(c&0x7F) << (7 * i)

		if c&0x80 == 0 {
			return value, nil
		}
	}
	return 0, fmt.Errorf("varint too long")
}

func (b *Buffer) ReadString() (string, error) {
	n, err := b.ReadVarInt()

	if err != nil {
		return "", err
	}
	if n > 65535 {
		return "", fmt.Errorf("string too long (%d)", n)
	}
	if n < 0 || n > b.r.Len() {
		return "", fmt.Errorf("requested %d bytes, %d available", n, b.r.Len())
	}

	buf := make([]byte, n)

	if _, err := io.ReadFull(b.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (b *Buffer) ReadUShort() (uint16, error) {
	var v uint16

	if err := binary.Read(b.r, binary.BigEndian, &v); err != nil {
		return 0, err
	}

	return v, nil
}

func (b *Buffer) ReadByteArray(maxLen int) ([]byte, error) {
	n, err := b.ReadVarInt()
	if err != nil {
		return nil, err
	}
	if n < 0 || n > maxLen {
		return nil, fmt.Errorf("byte array length %d out of bounds (max %d)", n, maxLen)
	}
	if n > b.r.Len() {
		return nil, fmt.Errorf("requested %d bytes, %d available", n, b.r.Len())
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(b.r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func WriteVarInt(buf *bytes.Buffer, v int) {
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if v == 0 {
			return
		}
	}
}

func WriteString(buf *bytes.Buffer, s string) {
	WriteVarInt(buf, len(s))
	buf.WriteString(s)
}

func WriteUShort(buf *bytes.Buffer, v uint16) {
	_ = binary.Write(buf, binary.BigEndian, v)
}

func WriteByteArray(buf *bytes.Buffer, b []byte) {
	WriteVarInt(buf, len(b))
	buf.Write(b)
}

// WriteFrame writes one length-prefixed frame.
// Format: VarInt(len(payload)) + payload.
func WriteFrame(w io.Writer, payload []byte) error {
	var prefix bytes.Buffer
	WriteVarInt(&prefix, len(payload))
	if _, err := w.Write(prefix.Bytes()); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
