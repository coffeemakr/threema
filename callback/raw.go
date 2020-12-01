package callback

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

const (
	fromStringLength    = 8
	toStringLength      = 8
	messageIDByteLength = 8
	nonceByteLength     = 24
	boxByteMaxLength    = 4000
	macByteLength       = 32
)

func getStringWithFixedLength(values url.Values, key string, requiredLength int) (string, error) {
	value := values.Get(key)
	if value == "" {
		return "", fmt.Errorf("Empty parameter %s", key)
	}
	if len(value) != requiredLength {
		return "", fmt.Errorf("Required length not met in %s", key)
	}
	return value, nil
}

func readRawValue(values url.Values) (value *RawCallbackValue, err error) {
	value = new(RawCallbackValue)
	value.From, err = getStringWithFixedLength(values, "from", fromStringLength)
	if err != nil {
		return
	}
	value.To, err = getStringWithFixedLength(values, "to", toStringLength)
	if err != nil {
		return
	}
	value.MessageID, err = getStringWithFixedLength(values, "messageId", messageIDByteLength*2)
	if err != nil {
		return
	}
	value.Date = values.Get("date")
	value.Nonce, err = getStringWithFixedLength(values, "nonce", nonceByteLength*2)
	if err != nil {
		return
	}

	value.Mac, err = getStringWithFixedLength(values, "mac", macByteLength*2)
	if err != nil {
		return
	}

	value.Box = values.Get("box")
	value.Nickname = values.Get("nickname")
	return
}

func ReadRawMessage(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeRequestError(w, fmt.Errorf("Invalid request method %s", r.Method))
			return
		}
		err := r.ParseForm()
		if err != nil {
			writeRequestError(w, err)
			return
		}
		println(r.Form.Encode())
		rawValue, err := readRawValue(r.Form)
		if err != nil {
			writeRequestError(w, err)
			return
		}
		newReuqest := r.WithContext(context.WithValue(r.Context(), rawThreemaValueKey, rawValue))
		h.ServeHTTP(w, newReuqest)
	})
}

type RawCallbackValue struct {
	// from sender identity (8 characters)
	From string

	// to your API identity (8 characters, usually starts with '*')
	To string

	// messageId message ID assigned by the sender (8 bytes, hex encoded)
	MessageID string

	// date message date set by the sender (UNIX timestamp)
	Date string

	// nonce used for encryption (24 bytes, hex encoded)
	Nonce string

	// box encrypted message data (max. 4000 bytes, hex encoded)
	Box string

	// mac Message Authentication Code (32 bytes, hex encoded, see below)
	Mac string

	// nickname public nickname of the sender, if set
	Nickname string
}

func (r *RawCallbackValue) parseMac() (mac []byte, err error) {
	return parseFixedSizeHexString(r.Mac, macByteLength)
}

func RawValueFromRequest(r *http.Request) *RawCallbackValue {
	return r.Context().Value(rawThreemaValueKey).(*RawCallbackValue)
}
