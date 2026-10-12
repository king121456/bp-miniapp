package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
)

type Cipher struct {
	aead cipher.AEAD
}

func New(dataKeyHex string) (*Cipher, error) {
	key, err := hex.DecodeString(dataKeyHex)
	if err != nil {
		return nil, err
	}
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

func (c *Cipher) Enc(s string) string {
	if s == "" {
		return ""
	}
	n := make([]byte, c.aead.NonceSize())
	_, _ = rand.Read(n)
	return base64.StdEncoding.EncodeToString(c.aead.Seal(n, n, []byte(s), nil))
}

func (c *Cipher) Dec(s string) string {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b) < c.aead.NonceSize() {
		return ""
	}
	p, err := c.aead.Open(nil, b[:c.aead.NonceSize()], b[c.aead.NonceSize():], nil)
	if err != nil {
		return ""
	}
	return string(p)
}
