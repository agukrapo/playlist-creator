package random

import (
	"crypto/rand"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func Name(length uint) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)

	for i := range length {
		b[i] = charset[int(b[i])%len(charset)]
	}

	return string(b)
}
