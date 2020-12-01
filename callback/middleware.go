package callback

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type key int

func parseDateString(rawDate string) (uint64, error) {
	return strconv.ParseUint(rawDate, 10, 64)
}

func parseFixedSizeHexString(rawValue string, byteLength int) ([]byte, error) {
	if len(rawValue) != (2 * byteLength) {
		return nil, fmt.Errorf("expected hex length %d but got %d", 2*byteLength, len(rawValue))
	}
	return hex.DecodeString(rawValue)
}

func handleMessage(message CallbackValue) {
	log.Println("Got message", message)
}

func writeRequestError(w http.ResponseWriter, err error) {
	log.Println("Failed parse request ", err)
	w.WriteHeader(http.StatusBadRequest)

}

const (
	rawThreemaValueKey    key = iota
	parsedThreemaValueKey key = iota
)

func ParsedValueFromRequest(r *http.Request) *CallbackValue {
	return r.Context().Value(parsedThreemaValueKey).(*CallbackValue)
}

func ReadParsedMessage(macVerification MacVerification, h http.Handler) http.Handler {
	return ReadRawMessage(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawValue := RawValueFromRequest(r)
		mac, err := rawValue.parseMac()
		if err != nil {
			writeRequestError(w, err)
			return
		}

		err = macVerification.verify(rawValue, mac)
		if err != nil {
			writeRequestError(w, err)
			return
		}
		binaryMessageID, err := parseFixedSizeHexString(rawValue.MessageID, messageIDByteLength)
		if err != nil {
			writeRequestError(w, fmt.Errorf("Failed to read message ID: %s", err))
			return
		}

		date, err := parseDateString(rawValue.Date)
		if err != nil {
			writeRequestError(w, err)
			return
		}
		nonce, err := parseFixedSizeHexString(rawValue.Nonce, nonceByteLength)
		if err != nil {
			writeRequestError(w, fmt.Errorf("Failed to read nonce: %s", err))
			return
		}

		box, err := hex.DecodeString(rawValue.Box)
		if err != nil {
			writeRequestError(w, err)
			return
		}

		value := &CallbackValue{
			From:      rawValue.From,
			To:        rawValue.To,
			MessageID: binaryMessageID,
			Date:      time.Unix(int64(date), 0),
			Nonce:     nonce,
			Box:       box,
			Mac:       mac,
			Nickname:  rawValue.Nickname,
		}

		newRequest := r.WithContext(context.WithValue(r.Context(), parsedThreemaValueKey, value))
		h.ServeHTTP(w, newRequest)
	}))
}
