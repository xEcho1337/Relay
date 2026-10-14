package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"net"
	"sync"
)

// Minecraft uses AES/CFB8 with the 16-byte shared secret as both key and
// initial IV. Go's standard library only provides CFB128, so CFB8 is
// implemented here one byte at a time using the feedback register.
type cfb8 struct {
	block cipher.Block
	fb    [aes.BlockSize]byte
}

func newCFB8(secret []byte) (*cfb8, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}
	c := &cfb8{block: block}
	copy(c.fb[:], secret)
	return c, nil
}

func (c *cfb8) cryptByte(in byte, encrypt bool) byte {
	var out [aes.BlockSize]byte
	c.block.Encrypt(out[:], c.fb[:])
	b := in ^ out[0]
	copy(c.fb[:], c.fb[1:])
	if encrypt {
		c.fb[aes.BlockSize-1] = b
	} else {
		c.fb[aes.BlockSize-1] = in
	}
	return b
}

// Conn encrypts writes and decrypts reads. Each direction has independent
// state, so one concurrent reader and one concurrent writer are safe.
type Conn struct {
	net.Conn
	enc *cfb8
	dec *cfb8
	rmu sync.Mutex
	wmu sync.Mutex
}

func NewConn(conn net.Conn, secret []byte) (*Conn, error) {
	enc, err := newCFB8(secret)
	if err != nil {
		return nil, err
	}
	dec, err := newCFB8(secret)
	if err != nil {
		return nil, err
	}
	return &Conn{Conn: conn, enc: enc, dec: dec}, nil
}

func (c *Conn) Read(b []byte) (int, error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	n, err := c.Conn.Read(b)
	for i := 0; i < n; i++ {
		b[i] = c.dec.cryptByte(b[i], false)
	}
	return n, err
}

func (c *Conn) Write(b []byte) (int, error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()

	tmp := make([]byte, len(b))
	for i, v := range b {
		tmp[i] = c.enc.cryptByte(v, true)
	}

	written := 0
	for written < len(tmp) {
		n, err := c.Conn.Write(tmp[written:])
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}
