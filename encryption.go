package threema

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/nacl/box"
)

type apiClient struct {
	secret string
}

const (
	cryptoBoxPublicKeyBytes = 32
	cryptoBoxSecretKeyBytes = 32
	cryptoBoxNonceBytes     = 24
)

var errWrongIdentityLength = errors.New("Wrong identity length")
var errWrongNonceLength = errors.New("Wrong identity length")
var errWrongSecretKeyLength = errors.New("Wrong identity length")
var errWrongPublicKeyLength = errors.New("Wrong identity length")

func checkIdentity(value string) error {
	if len(value) != 8 {
		return errWrongIdentityLength
	}
	return nil
}

type threemaEncryption struct {
	secretKey *[32]byte
}

func (e *threemaEncryption) encrypt(content []byte, publicKey *[32]byte) (*Box, error) {
	nonce, err := CreateNonce()
	if err != nil {
		return nil, err
	}
	result, err := encrypt(content, nonce, publicKey, e.secretKey)
	if err != nil {
		return nil, err
	}

	return &Box{
		Nonce:   nonce[:],
		Message: result,
	}, nil

}

func encrypt(m []byte, n *[24]byte, pk *[32]byte, sk *[32]byte) (c []byte, err error) {
	var result []byte
	result = box.Seal(result, m, n, pk, sk)
	return result, nil
}

func ReadHexSecretKey(hexSecretKey string) (*[32]byte, error) {
	keyBytes, err := hex.DecodeString(hexSecretKey)
	if err != nil {
		return nil, err
	}
	return checkSecKey(keyBytes)
}

func ReadHexPublicKey(hexPublicKey string) (*[32]byte, error) {
	keyBytes, err := hex.DecodeString(hexPublicKey)
	if err != nil {
		return nil, err
	}
	return checkPubKey(keyBytes)
}

func ThreemaEncryption(secret string) (*threemaEncryption, error) {
	secretKey, err := ReadHexSecretKey(secret)
	if err != nil {
		return nil, err
	}
	return &threemaEncryption{
		secretKey: secretKey,
	}, nil
}

func CreateNonce() (*[24]byte, error) {
	var nonce [24]byte
	_, err := rand.Read(nonce[:])
	return &nonce, err
}

type SecretKey [32]byte
type PublicKey [32]byte
type Nonce [24]byte

func checkPubKey(pk []byte) (*[32]byte, error) {
	if len(pk) != cryptoBoxPublicKeyBytes {
		return nil, errors.New("Invalid public key length")
	}
	var key [32]byte
	copy(key[:], pk[:32])
	return &key, nil
}

func checkSecKey(sk []byte) (*[32]byte, error) {
	if len(sk) != cryptoBoxPublicKeyBytes {
		return nil, errWrongSecretKeyLength
	}
	var key [32]byte
	copy(key[:], sk[:32])
	return &key, nil
}

type Box struct {
	Nonce   []byte
	Message []byte
}

func (e *threemaEncryption) EncryptText(message string, publicKey *[32]byte) (*Box, error) {
	plaintextBytes, err := PackTextMessage(message)
	if err != nil {
		return nil, err
	}
	return e.encrypt(plaintextBytes, publicKey)
}
