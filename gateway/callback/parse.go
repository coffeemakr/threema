package callback

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/coffeemakr/threema/gateway"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	fromStringLength    = 8
	toStringLength      = 8
	messageIDByteLength = 8
	boxByteMaxLength    = 4000
	macByteLength       = 32
)

func parseDateString(rawDate string) (result time.Time, err error) {
	var timestamp int64
	timestamp, err = strconv.ParseInt(rawDate, 10, 64)
	if err != nil {
		return
	}
	result = time.Unix(timestamp, 0)
	return
}

func parseMac(rawValue string) ([]byte, error) {
	if len(rawValue) != (2 * macByteLength) {
		return nil, fmt.Errorf("expected mac length %d (in hex) but got %d", 2 * macByteLength, len(rawValue))
	}
	return hex.DecodeString(rawValue)
}

func getStringWithFixedLength(values url.Values, key string, requiredLength int) (string, error) {
	value := values.Get(key)
	if value == "" {
		return "", fmt.Errorf("empty parameter %s", key)
	}
	if len(value) != requiredLength {
		return "", fmt.Errorf("required length not met in %s", key)
	}
	return value, nil
}

// read message from a request to the callback.
// This checks the request method and the mac to determine if the message is valid.
func ReadMessage(r *http.Request, apiSecret string) (message *EncryptedMessage, err error) {
	if r.Method != http.MethodPost {
		return nil, fmt.Errorf("invalid request method %s", r.Method)
	}
	calculatedMac := hmac.New(sha256.New, []byte(apiSecret))

	if err = r.ParseForm(); err != nil {
		return
	}
	message = new(EncryptedMessage)
	{
		message.From, err = getStringWithFixedLength(r.Form, "from", fromStringLength)
		calculatedMac.Write([]byte(message.From))
		if err != nil {
			return
		}
	}
	{
		message.To, err = getStringWithFixedLength(r.Form, "to", toStringLength)
		if err != nil {
			return
		}
		calculatedMac.Write([]byte(message.To))
	}
	{
		var rawMessageID string
		rawMessageID, err = getStringWithFixedLength(r.Form, "messageId", messageIDByteLength*2)
		if err != nil {
			return
		}
		calculatedMac.Write([]byte(rawMessageID))
		message.MessageID, err  = gateway.ReadMessageIDFromHex(rawMessageID)
		if err != nil {
			err = fmt.Errorf("failed to read message id: %s", err)
			return
		}
	}
	{
		rawDate := r.Form.Get("date")
		calculatedMac.Write([]byte(rawDate))
		message.Date, err = parseDateString(rawDate)
		if err != nil {
			return
		}
	}
	{
		rawNonce := r.Form.Get("nonce")
		message.Nonce, err = gateway.ReadHexNonce(rawNonce)
		if err != nil {
			return
		}
		calculatedMac.Write([]byte(rawNonce))
	}


	rawBox := r.Form.Get("box")
	calculatedMac.Write([]byte(rawBox))


	// nick name is not included in the mac, so it could be forged
	message.Nickname = r.Form.Get("nickname")
	if len(message.Nickname) > 32 {
		message.Nickname = message.Nickname[:32]
	}

	{
		var mac []byte
		mac, err = parseMac(r.Form.Get("mac"))
		if err != nil {
			return
		}
		if !hmac.Equal(calculatedMac.Sum(nil), mac) {
			err = errors.New("mac does not match")
		}
	}


	// Box is only decoded after mac verification, because it has a various length
	message.Box, err = hex.DecodeString(rawBox)
	return
}
