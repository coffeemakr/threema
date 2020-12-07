package gateway

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)


type Client struct {
	Secret         string
	ID             string
	Client *http.Client
}

func (c *Client) client() *http.Client{
	if c.Client != nil {
		return c.Client
	}
	return http.DefaultClient
}

var (
	ErrIDNotFound    = errors.New("threema identity not found")
	ErrBadSecret     = errors.New("api secret or identity is incorrect")
	ErrRequestFailed = errors.New("request failed")
	ErrInvalidRecipient = errors.New("recipient identity is invalid or the account is not set up for end-to-end mode")
	ErrMessageTooLong = errors.New("message is too long")
	ErrBlobTooBig = errors.New("blob is too big")
	ErrInternalServerError = errors.New("temporary server error")
	ErrMissingCredits = errors.New("missing credits")
)

// Lookup the public key of the Threema identity.
// If the identity doesn't exist, nil and ErrIDNotFound are returned.
func (c *Client) LookupPublicKey(threemaID string) (pk *PublicKey, err error) {
	if err = checkIdentity(threemaID); err != nil {
		return
	}
	response, err := c.client().Get(fmt.Sprintf("https://msgapi.threema.ch/pubkeys/%s?from=%s&secret=%s",
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
			if closeErr := response.Body.Close(); closeErr != nil && err == nil{
				err = closeErr
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
	resp, err = c.client().PostForm("https://msgapi.threema.ch/send_e2e",
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
			bodyBytes, err = ioutil.ReadAll(resp.Body)
			if err == nil {
				messageId = string(bodyBytes[:])
			}
			if closeErr := resp.Body.Close(); closeErr != nil && err == nil{
				err = closeErr
			}
		}
	case http.StatusBadRequest:
		err = ErrInvalidRecipient
	case http.StatusRequestEntityTooLarge:
		err = ErrMessageTooLong
	case http.StatusUnauthorized:
		err = ErrBadSecret
	case http.StatusInternalServerError:
		err = ErrInternalServerError
	case 402:
		err = ErrMissingCredits
	default:
		err = ErrRequestFailed
	}
	return
}


func randomBoundary() string {
	var length = 32
	charSet := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567889-_")
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		random := rand.Intn(len(charSet))
		randomChar := charSet[random]
		result[i] = randomChar
	}
	return string(result)
}

func transformToMultipart(fileReader io.Reader) (string, io.Reader) {
	boundary := randomBoundary()
	fileFormat := "--%s\r\nContent-Disposition: form-data; name=\"blob\"\r\n\"Content-type: application/octet-stream\"\r\n\r\n"
	filePart := fmt.Sprintf(fileFormat, boundary)
	bodyBottom := fmt.Sprintf("\r\n--%s--\r\n", boundary)
	body := io.MultiReader(strings.NewReader(filePart), fileReader, strings.NewReader(bodyBottom))
	contentType := fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
	return contentType, body
}

func readBlobID(reader io.Reader) (blobID *BlobID, err error){
	var bodyBytes = make([]byte, 32)
	var bytesRead int
	bytesRead, err = io.ReadAtLeast(reader, bodyBytes, 32)
	if err != nil {
		return
	}
	if bytesRead != 32 {
		err = errors.New("received invalid response")
		return
	}
	var bytesWritten int
	blobID = new([16]byte)
	bytesWritten, err = hex.Decode(blobID[:], bodyBytes)
	if err == nil && bytesWritten != 16 {
		err = errors.New("received invalid response")
	}
	return
}

// Send the message and returns the message ID
func (c *Client) UploadBlob(blob []byte) (blobID *BlobID, err error) {
	var resp *http.Response

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("blob", "blob")
	if err != nil {
		return
	}
	_, err = part.Write(blob)
	if err != nil {
		return
	}
	err = writer.Close()
	if err != nil {
		return
	}
	//contentType, requestBody := transformToMultipart(blobBody)
	request, err := http.NewRequest("POST", "https://msgapi.threema.ch/upload_blob", body)
	if err != nil {
		return
	}
	q := request.URL.Query()
	q.Add("secret", c.Secret)
	q.Add("from", c.ID)
	request.URL.RawQuery = q.Encode()

	request.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err = c.client().Do(request)
	if err != nil {
		return
	}

	switch resp.StatusCode {
	case http.StatusOK:
		{
			blobID, err = readBlobID(resp.Body)
			if closeErr := resp.Body.Close(); closeErr != nil && err == nil{
				err = closeErr
			}
		}
	case http.StatusBadRequest:
		err = ErrInvalidRecipient
	case http.StatusRequestEntityTooLarge:
		err = ErrBlobTooBig
	case http.StatusUnauthorized:
		err = ErrBadSecret
	case http.StatusInternalServerError:
		err = ErrInternalServerError
	case 402:
		err = ErrMissingCredits
	default:
		err = ErrRequestFailed
	}
	return
}
