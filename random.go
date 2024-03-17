package Artifex

import (
	"math/rand"
	"time"
)

func GenerateRandomCode(length int) string {
	rand.Seed(time.Now().UnixNano())
	charset := "abcdefghijkmnpqrstuvwxyz23456789ABCDEFGHIJKMNPQRSTUVWXYZ"
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}
