package callback

import "time"

// CallbackValue is the received content from the Threema server
type CallbackValue struct {
	// from sender identity (8 characters)
	From string

	// to your API identity (8 characters, usually starts with '*')
	To string

	// messageId message ID assigned by the sender (8 bytes, hex encoded)
	MessageID []byte

	// date message date set by the sender (UNIX timestamp)
	Date time.Time

	// nonce used for encryption (24 bytes, hex encoded)
	Nonce []byte

	// box encrypted message data (max. 4000 bytes, hex encoded)
	Box []byte

	// mac Box Authentication Code (32 bytes, hex encoded, see below)
	Mac []byte

	// nickname public nickname of the sender, if set
	Nickname string
}
