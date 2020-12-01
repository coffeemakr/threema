package callback

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
)

type MacVerification interface {
	verify(*RawCallbackValue, []byte) error
}

type noVerification struct {
}

func (n *noVerification) verify(*RawCallbackValue, []byte) error {
	return nil
}

var NoMacVerification MacVerification = &noVerification{}

type correctMacVerfication struct {
	apiSecret string
}

func (v *correctMacVerfication) verify(r *RawCallbackValue, mac []byte) (err error) {
	calculatedMac := hmac.New(sha256.New, []byte(v.apiSecret))
	calculatedMac.Write([]byte(r.From))
	calculatedMac.Write([]byte(r.To))
	calculatedMac.Write([]byte(r.MessageID))
	calculatedMac.Write([]byte(r.Date))
	calculatedMac.Write([]byte(r.Nonce))
	calculatedMac.Write([]byte(r.Box))
	if !hmac.Equal(calculatedMac.Sum(nil), []byte(mac)) {
		err = errors.New("mac does not match")
	}
	return
}

// CorrectMacVerification calculates the mac with the secret.
func CorrectMacVerification(secret string) MacVerification {
	return &correctMacVerfication{secret}
}
