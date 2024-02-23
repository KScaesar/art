package Artifex

import (
	"bytes"
	"crypto/aes"
)

func EncryptECB(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrorJoin3rdPartyWithMsg(ErrInvalidParameter, err, "new cipher from key")
	}

	blockSize := block.BlockSize()
	plaintext = pad(plaintext, blockSize)

	ciphertext := make([]byte, len(plaintext))

	for i := 0; i < len(plaintext); i += blockSize {
		block.Encrypt(ciphertext[i:i+blockSize], plaintext[i:i+blockSize])
	}

	return ciphertext, nil
}

func pad(plaintext []byte, blockSize int) []byte {
	padding := blockSize - (len(plaintext) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(plaintext, padText...)
}

func DecryptECB(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrorJoin3rdPartyWithMsg(ErrInvalidParameter, err, "new cipher from key")
	}

	blockSize := block.BlockSize()
	if len(ciphertext)%blockSize != 0 {
		return nil, ErrorWrapWithMessage(ErrInvalidParameter, "ciphertext is not a multiple of the block size")
	}

	plaintext := make([]byte, len(ciphertext))

	for i := 0; i < len(ciphertext); i += blockSize {
		block.Decrypt(plaintext[i:i+blockSize], ciphertext[i:i+blockSize])
	}

	return unPad(plaintext), nil
}

func unPad(data []byte) []byte {
	padding := int(data[len(data)-1])
	return data[:len(data)-padding]
}
