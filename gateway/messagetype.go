package gateway

type MessageType byte

const (
	TypeText            MessageType = 0x01
	TypeImage           MessageType = 0x02
	TypeLocation        MessageType = 0x10
	TypeVoice           MessageType = 0x14
	TypePoll            MessageType = 0x15
	TypeVote            MessageType = 0x16
	TypeFile            MessageType = 0x17
	TypeGroupText       MessageType = 0x41
	TypeGroupImage      MessageType = 0x43
	TypeGroupFile       MessageType = 0x46
	TypeAddedToGroup    MessageType = 0x4A
	TypeGroupCreated    MessageType = 0x4B
	TypeDeliveryReceipt MessageType = 0x80
)

type DeliveryReceiptType byte

const (
	DeliveryReceived     DeliveryReceiptType = 0x01
	DeliveryRead         DeliveryReceiptType = 0x02
	DeliveryAcknowledged DeliveryReceiptType = 0x03
	DeliveryDeclined     DeliveryReceiptType = 0x04
)
