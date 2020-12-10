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
	"strings"
)

const (
	blobIdBytes    = 16
	messageIdBytes = 8
	groupIdBytes   = 8
)

type BlobID = [blobIdBytes]byte
type MessageID = [messageIdBytes]byte
type GroupID [groupIdBytes]byte

func ReadMessageIDFromHex(hexValue string) (*MessageID, error) {
	messageID := new(MessageID)
	decoded, err := hex.Decode(messageID[:], []byte(hexValue))
	if err == nil && decoded != messageIdBytes {
		err = errWrongMessageIdLength
	}
	return messageID, err
}

type Message interface {
	Type() MessageType
	PackContent() []byte
	Unpack([]byte) error
}

func QuoteText(sender string, quote string, response string) string {
	lines := strings.Split(quote, "\n")
	result := strings.Join(lines, "\n> ")
	return "> " + sender + result + "\n" + response
}

// Returns a padding with a length between 1 and 255 (inclusive).
func RandomPadding() []byte {
	max := big.NewInt(254)
	randomNumber, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}
	padLen := int(randomNumber.Int64()) + 1
	padding := make([]byte, 1, padLen)
	padding[0] = byte(padLen)
	padding = bytes.Repeat(padding, padLen)
	return padding
}

type TextMessage struct {
	Content []byte
}

func (t *TextMessage) Type() MessageType {
	return TypeText
}

func (t *TextMessage) PackContent() []byte {
	return t.Content
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

func (i *ImageMessage) Type() MessageType {
	return TypeImage
}

func (i *ImageMessage) Unpack(content []byte) error {
	if len(content) != (blobIdBytes + cryptoBoxNonceBytes + 4) {
		return fmt.Errorf("invalid image message size %d != %d", len(content), blobIdBytes+cryptoBoxNonceBytes+4)
	}
	i.Nonce = new(Nonce)
	i.BlobID = new(BlobID)
	copy(i.BlobID[:], content[:blobIdBytes])
	i.Size = binary.LittleEndian.Uint32(content[blobIdBytes:])
	copy(i.Nonce[:], content[blobIdBytes+4:])
	return nil
}

func (i *ImageMessage) PackContent() (result []byte) {
	result = make([]byte, blobIdBytes+4, blobIdBytes+4+cryptoBoxNonceBytes)
	copy(result, i.BlobID[:])
	binary.LittleEndian.PutUint32(result[blobIdBytes:], i.Size)
	result = append(result, i.Nonce[:]...)
	return result
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

func (f *FileMessage) Type() MessageType {
	return TypeFile
}

func (f *FileMessage) PackContent() []byte {
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
	content, err := json.Marshal(jsonFile)
	if err != nil {
		panic(err)
	}
	return content
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
	DeliveryType DeliveryReceiptType
	MessageIDs   []*MessageID
}

func (d *DeliveryReceiptMessage) Type() MessageType {
	return TypeDeliveryReceipt
}

func (d *DeliveryReceiptMessage) PackContent() []byte {
	content := make([]byte, 1+(messageIdBytes*len(d.MessageIDs)))
	content = append(content, byte(d.DeliveryType))
	for _, messageId := range d.MessageIDs {
		content = append(content, messageId[:]...)
	}
	return content
}

func (d *DeliveryReceiptMessage) Unpack(content []byte) error {
	if len(content) < 9 || ((len(content)-1)%8 != 0) {
		return errors.New("invalid delivery message length")
	}
	if content[0] > 0x04 || content[0] == 0 {
		return errors.New("invalid delivery message type")
	}
	numberOfMessageIds := (len(content) - 1) / 8
	d.MessageIDs = make([]*MessageID, 0, numberOfMessageIds)
	d.DeliveryType = DeliveryReceiptType(content[0])
	for i := 1; i < len(content); i += 8 {
		messageId := new(MessageID)
		copy(messageId[:], content[i:i+8])
		d.MessageIDs = append(d.MessageIDs, messageId)
	}
	return nil
}

type VoiceMessage struct {
	// The length of the voice message in seconds
	Seconds   uint16
	// The ID of the blob
	BlobID    *BlobID
	// Size of the blob size
	Size      uint32
	// SharedKey of the blob
	SharedKey *SharedKey
}

func (v *VoiceMessage) Type() MessageType {
	return TypeVoice
}

func (v *VoiceMessage) Unpack(content []byte) error {
	v.Seconds = binary.LittleEndian.Uint16(content)
	v.BlobID = new(BlobID)
	content = content[2:]
	copy(v.BlobID[:], content[:blobIdBytes])
	content = content[blobIdBytes:]
	v.Size = binary.LittleEndian.Uint32(content)
	content = content[4:]
	v.SharedKey = new(SharedKey)
	copy(v.SharedKey[:], content[:cryptoBoxSharedKeyBytes])
	return nil
}

func (v *VoiceMessage) PackContent() []byte {
	content := make([]byte, 6+blobIdBytes+cryptoBoxSharedKeyBytes)
	binary.LittleEndian.PutUint16(content, v.Seconds)
	copy(content[2:], v.BlobID[:])
	binary.LittleEndian.PutUint32(content[2+blobIdBytes:], v.Size)
	copy(content[6+blobIdBytes:], v.SharedKey[:])
	return content
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

type OtherMessage struct {
	MessageType MessageType
	Content     []byte
}

func (o *OtherMessage) Type() MessageType {
	return o.MessageType
}

func (o *OtherMessage) PackContent() []byte {
	return o.Content
}

func (o *OtherMessage) Unpack(content []byte) error {
	o.Content = content
	return nil
}

func (o *OtherMessage) String() string {
	return fmt.Sprintf("OtherMessage type=%d, content=%s", o.Type, string(o.Content))
}

type GroupTextMessage struct {
	SenderID string
	GroupID  *GroupID
	Content  string
}

func (g *GroupTextMessage) Type() MessageType {
	return TypeGroupText
}

func (g *GroupTextMessage) Unpack(content []byte) error {
	g.SenderID = string(content[:8])
	g.GroupID = new(GroupID)
	copy(g.GroupID[:], content[8:16])
	g.Content = string(content[16:])
	return nil
}

func (g *GroupTextMessage) PackContent() []byte {
	result := make([]byte, 0, len(g.GroupID)+len(g.SenderID)+len(g.Content))
	result = append(result, []byte(g.SenderID)...)
	result = append(result, g.GroupID[:]...)
	result = append(result, []byte(g.Content)...)
	return result
}

func PackMessage(message Message) []byte {
	content := message.PackContent()
	padding := RandomPadding()
	result := make([]byte, 0, 1+len(content)+len(padding))
	result = append(result, byte(message.Type()))
	result = append(result, content...)
	return append(result, padding...)
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
	case TypeVoice:
		return unpackMessage(&VoiceMessage{}, content[1:])
	default:
		return &OtherMessage{
			MessageType: MessageType(content[0]),
			Content:     content[1:],
		}, nil
	}
}
