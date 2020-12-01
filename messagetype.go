package threema

type MessageType byte

const (
	TypeText            MessageType = 0x01
	TypeImage           MessageType = 0x02
	TypeFile            MessageType = 0x17
	TypeDeliveryReceipt MessageType = 0x80
)
