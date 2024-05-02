package art

import (
	cryptoRand "crypto/rand"

	"github.com/oklog/ulid/v2"
)

func GenerateRandomCode(length int) string {
	id := GenerateUlid()
	n := len(id)
	return id[n-length:]
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
