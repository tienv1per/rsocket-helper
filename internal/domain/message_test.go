package domain

import (
	"testing"
)

func TestNewTextMessage(t *testing.T) {
	payload := []byte("hello world")
	msg := NewTextMessage(payload)

	if msg.Type != MessageTypeText {
		t.Errorf("expected type to be Text, got %v", msg.Type)
	}
	if string(msg.Payload) != string(payload) {
		t.Errorf("expected payload to be %s, got %s", payload, msg.Payload)
	}
}

func TestNewBinaryMessage(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	msg := NewBinaryMessage(payload)

	if msg.Type != MessageTypeBinary {
		t.Errorf("expected type to be Binary, got %v", msg.Type)
	}
	if len(msg.Payload) != len(payload) {
		t.Errorf("expected payload length to be %d, got %d", len(payload), len(msg.Payload))
	}
	for i, b := range payload {
		if msg.Payload[i] != b {
			t.Errorf("expected payload[%d] to be %d, got %d", i, b, msg.Payload[i])
		}
	}
}

func TestMessageTypeString(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		expected string
	}{
		{MessageTypeText, "Text"},
		{MessageTypeBinary, "Binary"},
		{MessageType(99), "Unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.msgType.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessageValidate(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
		wantErr error
	}{
		{
			name: "valid text message",
			message: &Message{
				Type:    MessageTypeText,
				Payload: []byte("hello"),
			},
			wantErr: nil,
		},
		{
			name: "valid binary message",
			message: &Message{
				Type:    MessageTypeBinary,
				Payload: []byte{0x01, 0x02},
			},
			wantErr: nil,
		},
		{
			name: "valid text message with empty payload",
			message: &Message{
				Type:    MessageTypeText,
				Payload: []byte{},
			},
			wantErr: nil,
		},
		{
			name: "invalid message type",
			message: &Message{
				Type:    MessageType(99),
				Payload: []byte("test"),
			},
			wantErr: ErrInvalidMessageType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageIsText(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		expected bool
	}{
		{MessageTypeText, true},
		{MessageTypeBinary, false},
	}

	for _, tt := range tests {
		t.Run(tt.msgType.String(), func(t *testing.T) {
			msg := &Message{Type: tt.msgType}
			if got := msg.IsText(); got != tt.expected {
				t.Errorf("IsText() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessageIsBinary(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		expected bool
	}{
		{MessageTypeText, false},
		{MessageTypeBinary, true},
	}

	for _, tt := range tests {
		t.Run(tt.msgType.String(), func(t *testing.T) {
			msg := &Message{Type: tt.msgType}
			if got := msg.IsBinary(); got != tt.expected {
				t.Errorf("IsBinary() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessageToOpcode(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected Opcode
	}{
		{"text message to text opcode", MessageTypeText, OpcodeText},
		{"binary message to binary opcode", MessageTypeBinary, OpcodeBinary},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{Type: tt.msgType}
			if got := msg.ToOpcode(); got != tt.expected {
				t.Errorf("ToOpcode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessageTypeHandling(t *testing.T) {
	// Test creating and validating different message types
	textMsg := NewTextMessage([]byte("test text"))
	if err := textMsg.Validate(); err != nil {
		t.Errorf("text message validation failed: %v", err)
	}
	if !textMsg.IsText() {
		t.Error("expected text message to be text")
	}
	if textMsg.IsBinary() {
		t.Error("expected text message not to be binary")
	}
	if textMsg.ToOpcode() != OpcodeText {
		t.Errorf("expected text message opcode to be Text, got %v", textMsg.ToOpcode())
	}

	binaryMsg := NewBinaryMessage([]byte{0x01, 0x02, 0x03})
	if err := binaryMsg.Validate(); err != nil {
		t.Errorf("binary message validation failed: %v", err)
	}
	if binaryMsg.IsText() {
		t.Error("expected binary message not to be text")
	}
	if !binaryMsg.IsBinary() {
		t.Error("expected binary message to be binary")
	}
	if binaryMsg.ToOpcode() != OpcodeBinary {
		t.Errorf("expected binary message opcode to be Binary, got %v", binaryMsg.ToOpcode())
	}
}
