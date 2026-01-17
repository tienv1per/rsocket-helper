package domain

import (
	"testing"
)

func TestNewFrame(t *testing.T) {
	payload := []byte("test payload")
	frame := NewFrame(OpcodeText, payload)

	if frame.FIN != true {
		t.Errorf("expected FIN to be true, got %v", frame.FIN)
	}
	if frame.Opcode != OpcodeText {
		t.Errorf("expected opcode to be Text, got %v", frame.Opcode)
	}
	if frame.PayloadLen != uint64(len(payload)) {
		t.Errorf("expected payload length to be %d, got %d", len(payload), frame.PayloadLen)
	}
	if string(frame.Payload) != string(payload) {
		t.Errorf("expected payload to be %s, got %s", payload, frame.Payload)
	}
	if frame.Masked {
		t.Error("expected frame to be unmasked")
	}
}

func TestOpcodeIsControl(t *testing.T) {
	tests := []struct {
		name     string
		opcode   Opcode
		expected bool
	}{
		{"Continuation is not control", OpcodeContinuation, false},
		{"Text is not control", OpcodeText, false},
		{"Binary is not control", OpcodeBinary, false},
		{"Close is control", OpcodeClose, true},
		{"Ping is control", OpcodePing, true},
		{"Pong is control", OpcodePong, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opcode.IsControl(); got != tt.expected {
				t.Errorf("IsControl() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOpcodeIsData(t *testing.T) {
	tests := []struct {
		name     string
		opcode   Opcode
		expected bool
	}{
		{"Continuation is data", OpcodeContinuation, true},
		{"Text is data", OpcodeText, true},
		{"Binary is data", OpcodeBinary, true},
		{"Close is not data", OpcodeClose, false},
		{"Ping is not data", OpcodePing, false},
		{"Pong is not data", OpcodePong, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opcode.IsData(); got != tt.expected {
				t.Errorf("IsData() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOpcodeString(t *testing.T) {
	tests := []struct {
		opcode   Opcode
		expected string
	}{
		{OpcodeContinuation, "Continuation"},
		{OpcodeText, "Text"},
		{OpcodeBinary, "Binary"},
		{OpcodeClose, "Close"},
		{OpcodePing, "Ping"},
		{OpcodePong, "Pong"},
		{Opcode(0xFF), "Unknown(0xFF)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.opcode.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFrameValidate(t *testing.T) {
	tests := []struct {
		name    string
		frame   *Frame
		wantErr error
	}{
		{
			name: "valid text frame",
			frame: &Frame{
				FIN:        true,
				Opcode:     OpcodeText,
				PayloadLen: 5,
				Payload:    []byte("hello"),
			},
			wantErr: nil,
		},
		{
			name: "valid binary frame",
			frame: &Frame{
				FIN:        true,
				Opcode:     OpcodeBinary,
				PayloadLen: 3,
				Payload:    []byte{0x01, 0x02, 0x03},
			},
			wantErr: nil,
		},
		{
			name: "valid ping frame",
			frame: &Frame{
				FIN:        true,
				Opcode:     OpcodePing,
				PayloadLen: 4,
				Payload:    []byte("ping"),
			},
			wantErr: nil,
		},
		{
			name: "invalid opcode",
			frame: &Frame{
				FIN:        true,
				Opcode:     Opcode(0x03),
				PayloadLen: 0,
				Payload:    []byte{},
			},
			wantErr: ErrInvalidOpcode,
		},
		{
			name: "reserved bit set",
			frame: &Frame{
				FIN:        true,
				RSV1:       true,
				Opcode:     OpcodeText,
				PayloadLen: 0,
				Payload:    []byte{},
			},
			wantErr: ErrReservedBitsSet,
		},
		{
			name: "control frame too large",
			frame: &Frame{
				FIN:        true,
				Opcode:     OpcodePing,
				PayloadLen: 126,
				Payload:    make([]byte, 126),
			},
			wantErr: ErrInvalidFrameStructure,
		},
		{
			name: "fragmented control frame",
			frame: &Frame{
				FIN:        false,
				Opcode:     OpcodeClose,
				PayloadLen: 10,
				Payload:    make([]byte, 10),
			},
			wantErr: ErrInvalidFrameStructure,
		},
		{
			name: "payload length mismatch",
			frame: &Frame{
				FIN:        true,
				Opcode:     OpcodeText,
				PayloadLen: 10,
				Payload:    []byte("short"),
			},
			wantErr: ErrInvalidFrameStructure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.frame.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFrameIsControlFrame(t *testing.T) {
	tests := []struct {
		name     string
		opcode   Opcode
		expected bool
	}{
		{"text frame is not control", OpcodeText, false},
		{"binary frame is not control", OpcodeBinary, false},
		{"close frame is control", OpcodeClose, true},
		{"ping frame is control", OpcodePing, true},
		{"pong frame is control", OpcodePong, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &Frame{Opcode: tt.opcode}
			if got := frame.IsControlFrame(); got != tt.expected {
				t.Errorf("IsControlFrame() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFrameIsDataFrame(t *testing.T) {
	tests := []struct {
		name     string
		opcode   Opcode
		expected bool
	}{
		{"text frame is data", OpcodeText, true},
		{"binary frame is data", OpcodeBinary, true},
		{"continuation frame is data", OpcodeContinuation, true},
		{"close frame is not data", OpcodeClose, false},
		{"ping frame is not data", OpcodePing, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &Frame{Opcode: tt.opcode}
			if got := frame.IsDataFrame(); got != tt.expected {
				t.Errorf("IsDataFrame() = %v, want %v", got, tt.expected)
			}
		})
	}
}
