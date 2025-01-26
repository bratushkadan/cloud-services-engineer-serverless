package entity

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
)

func Id32() string {
	return Id(32)
}

func Id(n int) string {
	idBytes := make([]byte, n)
	_, _ = rand.Read(idBytes)
	dst := make([]byte, base32.StdEncoding.EncodedLen(len(idBytes)))
	base32.StdEncoding.Encode(dst, idBytes)

	return string(bytes.ToLower(bytes.TrimRight(dst, "=")))
}
