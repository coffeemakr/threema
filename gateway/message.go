package gateway

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
)


const (
	blobIdBytes = 16
	messageIdBytes = 8
)

type BlobID = [blobIdBytes]byte
type MessageID = [messageIdBytes]byte

func ReadMessageIDFromHex(hexValue string) (*MessageID, error) {
	messageID := new(MessageID)
	decoded, err := hex.Decode(messageID[:], []byte(hexValue))
	if err == nil && decoded != messageIdBytes {
		err = errWrongMessageIdLength
	}
	return messageID, err
}

type Message interface {
	Pack() ([]byte, error)
	Unpack([]byte) error
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

func (t *TextMessage) Pack() ([]byte, error) {
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

func (t *TextMessage) Unpack(content []byte) error {
	t.Content = content
	return nil
}

type ImageMessage struct {
	BlobID *BlobID
	Size   uint32
	Nonce  *Nonce
}

func (i *ImageMessage) Unpack(content []byte) error {
	if len(content) != (blobIdBytes + cryptoBoxNonceBytes + 4) {
		return fmt.Errorf("invalid image message size %d != %d", len(content), blobIdBytes + cryptoBoxNonceBytes + 4)
	}
	i.Nonce = new(Nonce)
	i.BlobID = new(BlobID)
	copy(i.BlobID[:], content[:blobIdBytes])
	i.Size = binary.LittleEndian.Uint32(content[blobIdBytes:])
	copy(i.Nonce[:], content[blobIdBytes + 4:])
	return nil
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
	binary.LittleEndian.PutUint32(result[len(i.BlobID)+1:], i.Size)
	result = append(result, i.Nonce[:]...)
	result = append(result, padding...)
	return result, nil
}

type FileMessage struct {
	FileID      *BlobID
	ThumbnailID *BlobID
	SharedKey   *SharedKey
	MimeType    string
	FileName    string
	FileSize    uint32
	Description string
}

func (f *FileMessage) Pack() ([]byte, error) {
	jsonFile := &jsonFile{
		FileBlobID:      hex.EncodeToString(f.FileID[:]),
		EncryptionKey:   hex.EncodeToString(f.SharedKey[:]),
		MimeType:        f.MimeType,
		FileName:        f.FileName,
		Size:            int64(f.FileSize),
		Version:         0,
		DescriptionText: f.Description,
	}
	if f.ThumbnailID != nil {
		jsonFile.ThumbnailBlobID = hex.EncodeToString(f.ThumbnailID[:])
	}
	return jsonFile.Pack()
}

type jsonFile struct {
	FileBlobID      string `json:"b"`
	ThumbnailBlobID string `json:"t,omitempty"`
	EncryptionKey   string `json:"k"`
	MimeType        string `json:"m"`
	FileName        string `json:"n,omitempty"`
	Size            int64  `json:"s"`
	Version         int16  `json:"i"`
	DescriptionText string `json:"d,omitempty"`
}

func (i *jsonFile) Pack() (result []byte, err error) {
	var padding []byte
	padding, err = RandomPadding()
	if err != nil {
		return nil, err
	}
	content, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	result = make([]byte, 0, 1+len(content)+len(padding))
	result = append(result, byte(TypeFile))
	result = append(result, content...)
	result = append(result, padding...)
	return
}

func (f *FileMessage) Unpack(content []byte) error {
	var jsonFileMessage jsonFile
	err := json.Unmarshal(content, &jsonFileMessage)
	if err != nil {
		return err
	}

	f.MimeType = jsonFileMessage.MimeType
	f.Description = jsonFileMessage.DescriptionText
	f.FileName = jsonFileMessage.FileName
	f.FileSize = uint32(jsonFileMessage.Size)

	f.SharedKey, err = ReadSharedKey(jsonFileMessage.EncryptionKey)
	if err != nil {
		return err
	}

	f.FileID, err = ReadBlobID(jsonFileMessage.FileBlobID)
	if err != nil {
		return err
	}
	if jsonFileMessage.ThumbnailBlobID != "" {
		f.ThumbnailID, err = ReadBlobID(jsonFileMessage.ThumbnailBlobID)
		if err != nil {
			return err
		}
	}
	return nil
}

type DeliveryReceiptMessage struct {
	Type DeliveryReceiptType
	MessageIDs []*MessageID
}

func (d DeliveryReceiptMessage) Pack() ([]byte, error) {
	padding, err := RandomPadding()
	if err != nil {
		return nil, err
	}
	content := make([]byte, 2 + 8 * len(d.MessageIDs) + len(padding))
	content[0] = byte(TypeDeliveryReceipt)
	for _, messageId := range d.MessageIDs {
		content = append(content, messageId[:]...)
	}
	content = append(content, padding...)
	return content, nil
}

func (d DeliveryReceiptMessage) Unpack(content []byte) error {
	if (len(content) - 1) % 8 != 0 {
		return errors.New("invalid delivery message length")
	}
	if content[0] > 0x04 || content[0] == 0 {
		return errors.New("invalid delivery message type")
	}
	numberOfMessageIds := (len(content) - 1) / 8
	d.MessageIDs = make([]*MessageID, 0, numberOfMessageIds)
	d.Type = DeliveryReceiptType(content[0])
	for i := 1; i < len(content); i += 8 {
		messageId := new(MessageID)
		copy(messageId[:], content[i:i+8])
		d.MessageIDs = append(d.MessageIDs, messageId)
	}
	return nil
}

func ReadSharedKey(hexKey string) (*SharedKey, error) {
	if len(hexKey) != (2 * cryptoBoxSharedKeyBytes) {
		return nil, errors.New("invalid shared key length")
	}
	key := new(SharedKey)
	_, err := hex.Decode(key[:], []byte(hexKey))
	return key, err
}

func removePadding(content []byte) ([]byte, error) {
	paddingLength := int(content[len(content)-1])
	if paddingLength == 0 {
		return nil, errors.New("padding is 0")
	}
	if len(content) < paddingLength {
		return nil, errors.New("padding is longer than message")
	}
	return content[:len(content)-paddingLength], nil
}

func unpackMessage(msg Message, content []byte) (Message, error) {
	err := msg.Unpack(content)
	return msg, err
}

func ReadMessage(content []byte) (Message, error) {
	var err error
	content, err = removePadding(content)
	if err != nil {
		return nil, err
	}
	if len(content) < 2 {
		return nil, errors.New("message has no content")
	}
	switch MessageType(content[0]) {
	case TypeText:
		return unpackMessage(&TextMessage{}, content[1:])
	case TypeFile:
		return unpackMessage(&FileMessage{}, content[1:])
	case TypeImage:
		return unpackMessage(&ImageMessage{}, content[1:])
	case TypeDeliveryReceipt:
		return unpackMessage(&DeliveryReceiptMessage{}, content[1:])
	default:
		return nil, errors.New("unknown message type")
	}
}
