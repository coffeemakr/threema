package gateway

import (
	"io/ioutil"
	"os"
)

type EncryptedClient struct {
	Client *Client
	EncryptionHelper EncryptionHelper
	PublicKeyStore *PublicKeyStore
}

func NewEncryptedClient(threemaId string, apiSecret string, secretKeyHex string) (*EncryptedClient, error) {
	err := checkIdentity(threemaId)
	if err != nil {
		return nil, err
	}
	encryptionHelper, err := NewEncryptionHelper(secretKeyHex)
	if err != nil {
		return nil, err
	}
	return &EncryptedClient{
		Client:           &Client{
			Secret: apiSecret,
			ID:     threemaId,
		},
		EncryptionHelper: encryptionHelper,
	}, nil
}

func (c *EncryptedClient) SendTextMessage(recipientID string, message string) (messageId string, err error) {
	return c.SendMessage(recipientID, TextMessage{ []byte(message) })
}

func (c *EncryptedClient) SendImage(recipientID string, filename string) (messageId string, err error) {
	var file *os.File
	var content []byte
	var publicKey *PublicKey

	if publicKey, err = c.LookupPublicKey(recipientID); err != nil {
		return
	}

	if file, err = os.Open(filename); err != nil {
		return
	}
	if content, err = ioutil.ReadAll(file); err != nil {
		_ = file.Close()
		return
	}
	if err = file.Close(); err != nil {
		return
	}
	imageMessage, err := c.EncryptionHelper.EncryptBytes(content, publicKey)
	if err != nil {
		return "", err
	}

	blobID, size, err := c.Client.UploadBlob(imageMessage.Box)
	if err != nil {
		return "", err
	}
	message := &ImageMessage{
		BlobID: blobID,
		Size:   uint32(size),
		Nonce:  imageMessage.Nonce,
	}
	return c.SendMessage(recipientID, message)
}

func (c *EncryptedClient) LookupPublicKey(recipientID string) (publicKey *PublicKey, err error) {
	if c.PublicKeyStore != nil {
		publicKey = (*c.PublicKeyStore).FetchPublicKey(recipientID)
	}
	if publicKey != nil {
		return
	}
	publicKey, err = c.Client.LookupPublicKey(recipientID)
	if err != nil {
		return
	}
	if c.PublicKeyStore != nil {
		err = (*c.PublicKeyStore).SavePublicKey(recipientID, publicKey)
	}
	return
}

func (c *EncryptedClient) SendMessage(recipientID string, message Message) (messageId string, err error) {
	var publicKey *PublicKey
	var encryptedMessage *EncryptedMessage
	if publicKey, err = c.LookupPublicKey(recipientID); err != nil {
		return
	}
	encryptedMessage, err = c.EncryptionHelper.EncryptMessage(message, publicKey)
	if err != nil {
		return
	}
	return c.Client.SendEncryptedMessage(recipientID, encryptedMessage)
}


type PublicKeyStore interface {
	FetchPublicKey(threemaID string) *PublicKey
	SavePublicKey(threemaID string, publicKey *PublicKey) error
}

type inMemoryStore map[string] *PublicKey

func (s inMemoryStore) FetchPublicKey(threemaID string) (pk *PublicKey) {
	return s[threemaID]
}
func (s inMemoryStore) SavePublicKey(threemaID string, publicKey *PublicKey) error {
	s[threemaID] = publicKey
	return nil
}

func NewInMemoryStore() PublicKeyStore {
	return inMemoryStore(make(map[string]*PublicKey))
}