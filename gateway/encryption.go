package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"golang.org/x/crypto/nacl/box"
)

const (
	cryptoBoxPublicKeyBytes = 32
	cryptoBoxSecretKeyBytes = 32
	cryptoBoxNonceBytes     = 24
)

var errWrongIdentityLength = errors.New("wrong identity length")
var errWrongNonceLength = errors.New("wrong identity length")
var errWrongSecretKeyLength = errors.New("wrong identity length")
var errWrongPublicKeyLength = errors.New("wrong identity length")

// The SecretKey contains the private key of the sender
type SecretKey = [cryptoBoxSecretKeyBytes]byte
// The PublicKey contains the public key of the receiver
type PublicKey = [cryptoBoxPublicKeyBytes]byte
// The Nonce
type Nonce = [cryptoBoxNonceBytes]byte

func checkIdentity(value string) error {
	if len(value) != 8 {
		return errWrongIdentityLength
	}
	return nil
}

type EncryptionHelper interface {
	EncryptMessage(message Message, publicKey *PublicKey) (*EncryptedMessage, error)
	EncryptBytes(content []byte, publicKey *PublicKey) (*EncryptedMessage, error)
	EncryptBytesWithNonce(content []byte, publicKey *PublicKey, nonce *Nonce) (*EncryptedMessage, error)
}

type encryptionHelper struct {
	secretKey *SecretKey
}

func (e encryptionHelper) EncryptBytesWithNonce(content []byte, publicKey *PublicKey, nonce *Nonce)  (message *EncryptedMessage, err error) {
	var boxBytes []byte
	if boxBytes, err = encrypt(content, nonce, publicKey, e.secretKey); err != nil {
		return nil, err
	}

	return &EncryptedMessage{
		Nonce: nonce,
		Box:   boxBytes,
	}, nil
}

func (e encryptionHelper) EncryptBytes(content []byte, publicKey *PublicKey) (message *EncryptedMessage, err error) {
	var nonce *Nonce
	if nonce, err = CreateNonce(); err != nil {
		return nil, err
	}
	return e.EncryptBytesWithNonce(content, publicKey, nonce)
}

func NewEncryptionHelper(secret string) (EncryptionHelper, error) {
	secretKey, err := ReadHexSecretKey(secret)
	if err != nil {
		return nil, err
	}
	return &encryptionHelper{
		secretKey: secretKey,
	}, nil
}

func encrypt(m []byte, n *Nonce, pk *PublicKey, sk *SecretKey) (c []byte, err error) {
	var result []byte
	result = box.Seal(result, m, n, pk, sk)
	return result, nil
}

func ReadHexSecretKey(hexSecretKey string) (*SecretKey, error) {
	keyBytes, err := hex.DecodeString(hexSecretKey)
	if err != nil {
		return nil, err
	}
	return checkSecKey(keyBytes)
}

func ReadHexPublicKey(hexPublicKey string) (*PublicKey, error) {
	keyBytes, err := hex.DecodeString(hexPublicKey)
	if err != nil {
		return nil, err
	}
	return checkPubKey(keyBytes)
}

func CreateNonce() (*Nonce, error) {
	var nonce [24]byte
	_, err := rand.Read(nonce[:])
	return &nonce, err
}

func checkPubKey(pk []byte) (*PublicKey, error) {
	if len(pk) != cryptoBoxPublicKeyBytes {
		return nil, errWrongPublicKeyLength
	}
	var key [32]byte
	copy(key[:], pk[:32])
	return &key, nil
}

func checkSecKey(sk []byte) (*SecretKey, error) {
	if len(sk) != cryptoBoxPublicKeyBytes {
		return nil, errWrongSecretKeyLength
	}
	var key [32]byte
	copy(key[:], sk[:32])
	return &key, nil
}

// The EncryptedMessage contains a message that can be decrypted by the recipient
type EncryptedMessage struct {
	// A random Nonce
	Nonce *Nonce
	// The encrypted Box that contains the actual message
	Box   []byte
}

func (e *encryptionHelper)  EncryptMessage(message Message, publicKey *PublicKey)  (*EncryptedMessage, error) {
	plaintextBytes, err := message.Pack()
	if err != nil {
		return nil, err
	}
	return e.EncryptBytes(plaintextBytes, publicKey)
}