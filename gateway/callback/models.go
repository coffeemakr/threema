package callback

import (
	"github.com/coffeemakr/threema/gateway"
	"time"
)

// EncryptedMessage is the received content from the Threema server
type EncryptedMessage struct {
	// from sender identity (8 characters)
	From string

	// to your API identity (8 characters, usually starts with '*')
	To string

	// messageId message ID assigned by the sender (8 bytes, hex encoded)
	MessageID *gateway.MessageID

	// date message date set by the sender (UNIX timestamp)
	Date time.Time

	// nonce used for encryption (24 bytes, hex encoded)
	Nonce *gateway.Nonce

	// box encrypted message data (max. 4000 bytes, hex encoded)
	Box []byte

	// mac Box Authentication Code (32 bytes, hex encoded, see below)
	Mac []byte

	// nickname public nickname of the sender, if set
	Nickname string
}
