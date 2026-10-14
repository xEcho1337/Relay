package crypt

import (
	"bytes"
	"testing"
)

func TestCFB8RoundTrip(t *testing.T) {
	secret := bytes.Repeat([]byte{0x07}, 16)
	plain := []byte("hello minecraft proxy relay test 12345!")

	enc, err := newCFB8(secret)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := newCFB8(secret)
	if err != nil {
		t.Fatal(err)
	}

	ciphered := make([]byte, len(plain))
	for i, b := range plain {
		ciphered[i] = enc.cryptByte(b, true)
	}
	if bytes.Equal(ciphered, plain) {
		t.Fatal("ciphertext equals plaintext")
	}

	back := make([]byte, len(ciphered))
	for i, b := range ciphered {
		back[i] = dec.cryptByte(b, false)
	}
	if !bytes.Equal(back, plain) {
		t.Fatalf("round trip failed: got %q want %q", back, plain)
	}
}
