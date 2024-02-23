package Artifex

import (
	"math/rand"
)

func GenerateRandomCode(length int) string {
	charset := "abcdefghijkmnpqrstuvwxyz23456789"
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}
