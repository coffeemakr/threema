package gateway

type MessageType byte

const (
	TypeText            MessageType = 0x01
	TypeImage           MessageType = 0x02
	TypeFile            MessageType = 0x17
	TypeDeliveryReceipt MessageType = 0x80
)

type DeliveryReceiptType byte

const (
	DeliveryReceived     DeliveryReceiptType = 0x01
	DeliveryRead         DeliveryReceiptType = 0x02
	DeliveryAcknowledged DeliveryReceiptType = 0x03
	DeliveryDeclined     DeliveryReceiptType = 0x04
)

