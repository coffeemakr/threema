package gateway

import (
	"crypto/rand"
	"golang.org/x/crypto/nacl/secretbox"
)

type SharedKey = [32]byte

func RandomSecretKey()  (*SharedKey, error) {
	key := new([32]byte)
	_, err := rand.Read(key[:])
	if err != nil {
		return nil, err
	}
	return key, nil
}


func EncryptWithSharedKey(content []byte, sharedKey *SharedKey, nonce *Nonce) []byte {
	return secretbox.Seal(nil, content, nonce, sharedKey)
}
