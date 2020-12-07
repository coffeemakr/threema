package gateway

import (
	"io"
	"io/ioutil"
	"mime"
	"os"
	"path"
)

type EncryptedClient struct {
	Client           *Client
	EncryptionHelper EncryptionHelper
	PublicKeyStore   PublicKeyStore
}

type nopKeyStore struct {}
func (nopKeyStore) FetchPublicKey(threemaID string) *PublicKey {
	return nil
}
func (nopKeyStore) SavePublicKey(threemaID string, publicKey *PublicKey) error {
	return nil
}

func (c *EncryptedClient) keystore() PublicKeyStore  {
	if c.PublicKeyStore == nil {
		return nopKeyStore{}
	}
	return c.PublicKeyStore
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
		Client: &Client{
			Secret: apiSecret,
			ID:     threemaId,
		},
		EncryptionHelper: encryptionHelper,
	}, nil
}

func (c *EncryptedClient) SendTextMessage(recipientID string, message string) (messageId string, err error) {
	return c.SendMessage(recipientID, TextMessage{[]byte(message)})
}

type BlobReference struct {
	BlobID *BlobID
	Size   uint32
	Nonce  *Nonce
}

var FileNonce = &[24]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
var ThumbnailNonce = &[24]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}

// Upload a plaintext blob to the gateway. nonce can be nil, for random nonce
func (c *EncryptedClient) UploadFile(reader io.Reader, sharedKey *SharedKey, nonce *Nonce) (*BlobReference, error) {
	var err error

	var content []byte
	if content, err = ioutil.ReadAll(reader); err != nil {
		return nil, err
	}

	box := EncryptWithSharedKey(content, sharedKey, nonce)
	blobID, err := c.Client.UploadBlob(box)
	return &BlobReference{
		BlobID: blobID,
		Size:   uint32(len(content)),
		Nonce:  nonce,
	}, nil
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
	size := uint32(len(content))
	if err = file.Close(); err != nil {
		return
	}
	imageMessage, err := c.EncryptionHelper.EncryptBytes(content, publicKey)
	if err != nil {
		return "", err
	}

	blobID, err := c.Client.UploadBlob(imageMessage.Box)
	if err != nil {
		return "", err
	}
	message := &ImageMessage{
		BlobID: blobID,
		Size:   size,
		Nonce:  imageMessage.Nonce,
	}
	return c.SendMessage(recipientID, message)
}

func (c *EncryptedClient) LookupPublicKey(recipientID string) (publicKey *PublicKey, err error) {
	if c.PublicKeyStore != nil {
		publicKey = c.keystore().FetchPublicKey(recipientID)
	}
	if publicKey != nil {
		return
	}
	publicKey, err = c.Client.LookupPublicKey(recipientID)
	if err != nil {
		return
	}
	if c.PublicKeyStore != nil {
		err = c.keystore().SavePublicKey(recipientID, publicKey)
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

type inMemoryStore map[string]*PublicKey

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

type File interface {
	Name() string
	Open() (io.ReadCloser, error)
	MimeType() string
	HasThumbnail() bool
	OpenThumbnail() (io.ReadCloser, error)
}

type FilePath struct {
	Path          string
	ThumbnailPath string
}

func (s FilePath) Open() (io.ReadCloser, error) {
	return os.Open(s.Path)
}

func (s FilePath) OpenThumbnail() (io.ReadCloser, error) {
	return os.Open(s.ThumbnailPath)
}

func (s FilePath) MimeType() string {
	return mime.TypeByExtension(path.Ext(s.Path))
}

func (s FilePath) Name() string  {
	return path.Base(s.Path)
}

func (s FilePath) HasThumbnail() bool {
	return s.ThumbnailPath != ""
}

func (c *EncryptedClient) PrepareFile(file File) (msg *FileMessage, err error) {
	var reader io.ReadCloser
	var blob *BlobReference
	var thumbnailBlodID *BlobID
	var sharedKey *SharedKey

	if sharedKey, err = RandomSecretKey(); err != nil {
		return
	}

	// Upload file
	if reader, err = file.Open(); err != nil {
		return
	}
	if blob, err = c.UploadFile(reader, sharedKey, FileNonce); err != nil {
		_ = reader.Close()
		return
	}
	if err = reader.Close(); err != nil {
		return
	}

	// Upload thumbnail
	if file.HasThumbnail() {
		var thumbnailBlob *BlobReference
		if reader, err = file.OpenThumbnail(); err != nil {
			return
		}
		if thumbnailBlob, err = c.UploadFile(reader, sharedKey, ThumbnailNonce); err != nil {
			_ = reader.Close()
			return
		}
		if err = reader.Close(); err != nil {
			return
		}
		thumbnailBlodID = thumbnailBlob.BlobID
	}

	mimeType := file.MimeType()
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return &FileMessage{
		FileID:      blob.BlobID,
		ThumbnailID: thumbnailBlodID,
		SharedKey:   sharedKey,
		MimeType:    mimeType,
		FileName:    file.Name(),
		FileSize:    blob.Size,
	}, nil
}

func (c *EncryptedClient) SendFile(recipientID string, file File, description string) (messageID string, err error) {
	var message *FileMessage

	// We lookup the public key first, because it doesn't use credits.
	if _, err = c.LookupPublicKey(recipientID); err != nil {
		return
	}

	message, err = c.PrepareFile(file)
	if err != nil {
		return
	}
	message.Description = description

	return c.SendMessage(recipientID, message)
}
