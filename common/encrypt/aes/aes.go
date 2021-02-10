package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"math/rand"
)

var (
	DefaultKey = []byte("default@cctest")
	keyPool    = []byte("fWk2sdXsMDd133fQ6faje38013X2K44iisf42f33d0d4dEfdFf440RE58foB28Zrerok5jl2kdzG9w43eDZqw7dfnT5364cdQ45dff4ga0dVn3fUddsSEah4Nd62zdIfWP2S4Wdh4f83Rd3uT5L9Upj32nPWgL6AO7df9dq8F0IwOe1")
)

type Key struct {
	K []byte
}

func MakeKey16() []byte {
	lenKeyPool := len(keyPool)
	if lenKeyPool < 16 {
		panic("keyPool too short")
	}

	r := rand.Intn(lenKeyPool - 15)
	return keyPool[r : r+16]
}

func Encrypt(origData, key []byte) ([]byte, error) {
	if key == nil {
		key = DefaultKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData = padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)

	return crypted, nil
}

func Decrypt(crypted, key []byte) ([]byte, error) {
	if key == nil {
		key = DefaultKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	if len(crypted)%blockSize != 0 {
		return nil, errors.New("decrypt data is not integer multiples of blocksize 16")
	}

	if blockSize > len(key) {
		return nil, fmt.Errorf("decrypt wrong blockSize(%d), key(%s)", blockSize, string(key))
	}

	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = unPadding(origData, blockSize)
	return origData, nil
}

func padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func unPadding(plantText []byte, blockSize int) []byte {
	length := len(plantText)
	unpadding := int(plantText[length-1])
	if unpadding >= length {
		return nil
	}
	return plantText[:(length - unpadding)]
}
