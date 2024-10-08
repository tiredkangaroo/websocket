package websocket

import "fmt"

// MessageType represents the possible types of messages.
type MessageType uint8

const (
	// MessageText is a UTF-8 encoded string.
	MessageText MessageType = 0
	// MessageBinary represents binary data.
	MessageBinary MessageType = 1
	// MessageClose represents a close handshake.
	MessageClose MessageType = 2
	// MessagePing represents a ping message from
	// the client.
	MessagePing MessageType = 3
	// MessagePong represents a pong message (usually
	// in return to a ping message).
	MessagePong MessageType = 4
)

// Message represents a WebSocket message.
type Message struct {
	Type MessageType
	Data []byte
}

// String returns the message as string formatted as:
// type: MessageType || data: MessageDataAsString
func (m Message) String() string {
	return fmt.Sprintf("type: %s || data: %s", m.Type.String(), m.Data)
}

// String returns the MessageType as a string.
func (t MessageType) String() string {
	switch t {
	case MessageText:
		return "MessageText"
	case MessageBinary:
		return "MessageBinary"
	case MessageClose:
		return "MessageClose"
	case MessagePing:
		return "MessagePing"
	case MessagePong:
		return "MessagePong"
	default:
		return "Unknown"
	}
}
