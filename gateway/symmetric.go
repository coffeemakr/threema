package gateway

import (
	"crypto/rand"
	"errors"
	"golang.org/x/crypto/nacl/secretbox"
)

const cryptoBoxSharedKeyBytes = 32
type SharedKey = [cryptoBoxSharedKeyBytes]byte

func RandomSecretKey()  (*SharedKey, error) {
	key := new(SharedKey)
	_, err := rand.Read(key[:])
	if err != nil {
		return nil, err
	}
	return key, nil
}

func EncryptWithSharedKey(content []byte, nonce *Nonce, sharedKey *SharedKey) []byte {
	return secretbox.Seal(nil, content, nonce, sharedKey)
}

func DecryptWithSharedSecret(content []byte, nonce *Nonce, sharedKey *SharedKey) ([]byte, error) {
	plaintext, ok := secretbox.Open(nil, content, nonce, sharedKey)
	if !ok {
		return nil, errors.New("decryption failed")
	}
	return plaintext, nil
}
