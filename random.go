package Artifex

import (
	cryptoRand "crypto/rand"
	mathRand "math/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

func GenerateRandomCode(length int) string {
	mathRand.Seed(time.Now().UnixNano())
	charset := "abcdefghijkmnpqrstuvwxyz23456789ABCDEFGHJKMNPQRSTUVWXYZ"
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[mathRand.Intn(len(charset))]
	}
	return string(code)
}

// https://github.com/oklog/ulid?tab=readme-ov-file#usage
var entropyPool = newPool(func() *ulid.MonotonicEntropy {
	return ulid.Monotonic(cryptoRand.Reader, 0)
})

func GenerateUlid() string {
	entropy := entropyPool.Get()
	id := ulid.MustNew(ulid.Now(), entropy)
	entropyPool.Put(entropy)
	return id.String()
}
