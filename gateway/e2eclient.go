package gateway

type EncryptedClient struct {
	Client *Client
	EncryptionHelper EncryptionHelper
	PublicKeyStore *PublicKeyStore
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
		Client:           &Client{
			Secret: apiSecret,
			ID:     threemaId,
		},
		EncryptionHelper: encryptionHelper,
	}, nil
}

func (c *EncryptedClient) SendTextMessage(recipientID string, textMessage string) (messageId string, err error) {
	var publicKey *PublicKey
	var message *EncryptedMessage
	if c.PublicKeyStore != nil {
		publicKey = (*c.PublicKeyStore).FetchPublicKey(recipientID)
	}
	if publicKey == nil {
		publicKey, err = c.Client.LookupPublicKey(recipientID)
		if err != nil {
			return
		}
		if c.PublicKeyStore != nil {
			err = (*c.PublicKeyStore).SavePublicKey(recipientID, publicKey)
			if err != nil {
				return
			}
		}
	}
	message, err = c.EncryptionHelper.EncryptText(textMessage, publicKey)
	if err != nil {
		return
	}
	return c.Client.SendEncryptedMessage(recipientID, message)
}


type PublicKeyStore interface {
	FetchPublicKey(threemaID string) *PublicKey
	SavePublicKey(threemaID string, publicKey *PublicKey) error
}

type inMemoryStore map[string] *PublicKey

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