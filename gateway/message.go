package gateway

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"math/big"
)

type Message interface {
	Pack() ([]byte, error)
}

// Returns a padding with a length between 1 and 255 (inclusive).
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

type TextMessage struct {
	Content []byte
}

func (t TextMessage) Pack() ([]byte, error) {
	padding, err := RandomPadding()
	if err != nil {
		return nil, err
	}
	content := []byte(t.Content)
	result := make([]byte, 1, len(content)+len(padding)+1)
	result[0] = byte(TypeText)
	result = append(result, content...)
	result = append(result, padding...)
	return result, nil
}

type BlobID = [16]byte

type ImageMessage struct {
	BlobID *BlobID
	Size uint32
	Nonce *Nonce
}

func (i *ImageMessage) Pack() (result []byte, err error) {
	var padding []byte
	padding, err = RandomPadding()
	if err != nil {
		return nil, err
	}
	result = make([]byte, len(i.BlobID)+5, len(i.BlobID)+5+len(i.Nonce)+len(padding))
	result[0] = byte(TypeImage)
	copy(result[1:], i.BlobID[:])
	binary.LittleEndian.PutUint32(result[len(i.BlobID) + 1:], i.Size)
	result = append(result, i.Nonce[:]...)
	result = append(result, padding...)
	return result, nil
}