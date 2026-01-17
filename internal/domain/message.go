package domain

import "fmt"

// MessageType represents the type of WebSocket message
type MessageType int

const (
	// MessageTypeText represents a text message
	MessageTypeText MessageType = iota
	// MessageTypeBinary represents a binary message
	MessageTypeBinary
)

// String returns the string representation of the message type
func (m MessageType) String() string {
	switch m {
	case MessageTypeText:
		return "Text"
	case MessageTypeBinary:
		return "Binary"
	default:
		return fmt.Sprintf("Unknown(%d)", int(m))
	}
}

// Message represents a WebSocket message
type Message struct {
	Type    MessageType // Message type (text or binary)
	Payload []byte      // Message payload
}

// NewTextMessage creates a new text message
func NewTextMessage(payload []byte) *Message {
	return &Message{
		Type:    MessageTypeText,
		Payload: payload,
	}
}

// NewBinaryMessage creates a new binary message
func NewBinaryMessage(payload []byte) *Message {
	return &Message{
		Type:    MessageTypeBinary,
		Payload: payload,
	}
}

// Validate checks if the message is valid
func (m *Message) Validate() error {
	// Check if message type is valid
	if m.Type != MessageTypeText && m.Type != MessageTypeBinary {
		return ErrInvalidMessageType
	}

	// Payload can be empty for some use cases, so we don't enforce non-empty

	return nil
}

// IsText returns true if this is a text message
func (m *Message) IsText() bool {
	return m.Type == MessageTypeText
}

// IsBinary returns true if this is a binary message
func (m *Message) IsBinary() bool {
	return m.Type == MessageTypeBinary
}

// ToOpcode converts the message type to the corresponding frame opcode
func (m *Message) ToOpcode() Opcode {
	switch m.Type {
	case MessageTypeText:
		return OpcodeText
	case MessageTypeBinary:
		return OpcodeBinary
	default:
		return OpcodeBinary // Default to binary
	}
}
