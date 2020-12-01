package threema

import (
	"bytes"
	"crypto/rand"
	"math/big"
)

const (
	blockSize = 255
)

func RandomPadding() ([]byte, error) {
	max := big.NewInt(254)
	randomNumber, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, err
	}
	padLen := int(randomNumber.Int64()) + 1
	padding := make([]byte, 1, padLen)
	padding[0] = byte(padLen)
	padding = bytes.Repeat(padding, padLen)
	return padding, nil
}

func PackTextMessage(message string) ([]byte, error) {
	messageBytes := []byte(message)
	padding, err := RandomPadding()
	if err != nil {
		return nil, err
	}

	result := make([]byte, 1, len(message)+len(padding)+1)
	result[0] = byte(TypeText)
	result = append(result, messageBytes...)
	result = append(result, padding...)
	return result, nil
}
