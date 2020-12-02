package gateway

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)


type Client struct {
	Secret         string
	ID             string
}

var (
	ErrIDNotFound    = errors.New("threema identity not found")
	ErrBadSecret     = errors.New("api secret or identity is incorrect")
	ErrRequestFailed = errors.New("request failed")
	ErrInvalidRecipient = errors.New("recipient identity is invalid or the account is not set up for end-to-end mode")
	ErrMessageTooLong = errors.New("message is too long")
	ErrInternalServerError = errors.New("temporary server error")
)

// Lookup the public key of the Threema identity.
// If the identity doesn't exist, nil and ErrIDNotFound are returned.
func (c *Client) LookupPublicKey(threemaID string) (pk *PublicKey, err error) {
	if err = checkIdentity(threemaID); err != nil {
		return
	}
	response, err := http.Get(fmt.Sprintf("https://msgapi.threema.ch/pubkeys/%s?from=%s&secret=%s",
		url.PathEscape(threemaID), url.QueryEscape(c.ID), url.QueryEscape(c.Secret)))
	if err != nil {
		return nil, err
	}
	switch response.StatusCode {
	case http.StatusOK:
		{
			body, err := ioutil.ReadAll(response.Body)
			if err == nil {
				pk, err = ReadHexPublicKey(string(body[:]))
			}
			if err == nil {
				err = response.Body.Close()
			}
		}
	case http.StatusNotFound:
		err = ErrIDNotFound
	case http.StatusUnauthorized:
		err = ErrBadSecret
	case http.StatusInternalServerError:
		err = ErrInternalServerError
	default:
		err = ErrRequestFailed
	}
	return
}

// Send the message and returns the message ID
func (c *Client) SendEncryptedMessage(to string, box *EncryptedMessage) (messageId string, err error) {
	var resp *http.Response
	resp, err = http.PostForm("https://msgapi.threema.ch/send_e2e",
		url.Values{"from": {c.ID},
			"to":     {to},
			"nonce":  {hex.EncodeToString((*box.Nonce)[:])},
			"box":    {hex.EncodeToString(box.Box)},
			"secret": {c.Secret}})
	if err != nil {
		return
	}
	switch resp.StatusCode {
	case http.StatusOK:
		{
			var bodyBytes []byte
			if resp.StatusCode != 200 {
				return "", fmt.Errorf("Reqeust failed: %s", resp.Status)
			}
			bodyBytes, err = ioutil.ReadAll(resp.Body)
			if err == nil {
				messageId = string(bodyBytes[:])
			}
			err = resp.Body.Close()
		}
	case http.StatusBadRequest:
		err = ErrInvalidRecipient
	case http.StatusRequestEntityTooLarge:
		err = ErrMessageTooLong
	case http.StatusUnauthorized:
		err = ErrBadSecret
	case http.StatusInternalServerError:
		err = ErrInternalServerError
	default:
		err = ErrRequestFailed
	}
	return
}
