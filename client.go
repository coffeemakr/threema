package threema

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type GatewayClient struct {
	Secret string
	ID     string
}

func (c *GatewayClient) LookupPublicKey(to string) (*[32]byte, error) {
	resp, err := http.Get(fmt.Sprintf("https://msgapi.threema.ch/pubkeys/%s?from=%s&secret=%s",
		url.PathEscape(to), url.QueryEscape(c.ID), url.QueryEscape(c.Secret)))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Reqeust failed: %s", resp.Status)
	}
	pubkey, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return ReadHexPublicKey(string(pubkey[:]))
}

func (c *GatewayClient) SendMessage(to string, box *Box) (string, error) {
	println(hex.EncodeToString(box.Nonce))
	resp, err := http.PostForm("https://msgapi.threema.ch/send_e2e",
		url.Values{"from": {c.ID},
			"to":     {to},
			"nonce":  {hex.EncodeToString(box.Nonce)},
			"box":    {hex.EncodeToString(box.Message)},
			"secret": {c.Secret}})
	log.Println(resp.Request)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Reqeust failed: %s", resp.Status)
	}
	messageId, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(messageId[:]), nil
}
