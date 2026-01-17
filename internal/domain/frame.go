package domain

import "fmt"

// Opcode represents the WebSocket frame opcode
type Opcode byte

// WebSocket frame opcodes as defined in RFC 6455
const (
	OpcodeContinuation Opcode = 0x0
	OpcodeText         Opcode = 0x1
	OpcodeBinary       Opcode = 0x2
	OpcodeClose        Opcode = 0x8
	OpcodePing         Opcode = 0x9
	OpcodePong         Opcode = 0xA
)

// IsControl returns true if the opcode is a control frame
func (o Opcode) IsControl() bool {
	return o >= 0x8
}

// IsData returns true if the opcode is a data frame
func (o Opcode) IsData() bool {
	return o <= 0x2
}

// String returns the string representation of the opcode
func (o Opcode) String() string {
	switch o {
	case OpcodeContinuation:
		return "Continuation"
	case OpcodeText:
		return "Text"
	case OpcodeBinary:
		return "Binary"
	case OpcodeClose:
		return "Close"
	case OpcodePing:
		return "Ping"
	case OpcodePong:
		return "Pong"
	default:
		return fmt.Sprintf("Unknown(0x%X)", byte(o))
	}
}

// Frame represents a WebSocket frame as defined in RFC 6455
type Frame struct {
	FIN        bool    // Final fragment flag
	RSV1       bool    // Reserved bit 1
	RSV2       bool    // Reserved bit 2
	RSV3       bool    // Reserved bit 3
	Opcode     Opcode  // Frame opcode
	Masked     bool    // Payload is masked
	PayloadLen uint64  // Payload length
	MaskingKey [4]byte // Masking key (if masked)
	Payload    []byte  // Payload data
}

// NewFrame creates a new frame with the given opcode and payload
func NewFrame(opcode Opcode, payload []byte) *Frame {
	return &Frame{
		FIN:        true,
		RSV1:       false,
		RSV2:       false,
		RSV3:       false,
		Opcode:     opcode,
		Masked:     false,
		PayloadLen: uint64(len(payload)),
		Payload:    payload,
	}
}

// Validate checks if the frame is valid according to RFC 6455
func (f *Frame) Validate() error {
	// Check if opcode is valid
	if !f.isValidOpcode() {
		return ErrInvalidOpcode
	}

	// Check if reserved bits are set (they should be 0 unless extensions are negotiated)
	if f.RSV1 || f.RSV2 || f.RSV3 {
		return ErrReservedBitsSet
	}

	// Control frames must have payload length <= 125
	if f.Opcode.IsControl() && f.PayloadLen > 125 {
		return ErrInvalidFrameStructure
	}

	// Control frames must not be fragmented
	if f.Opcode.IsControl() && !f.FIN {
		return ErrInvalidFrameStructure
	}

	// Payload length must match actual payload
	if uint64(len(f.Payload)) != f.PayloadLen {
		return ErrInvalidFrameStructure
	}

	return nil
}

// isValidOpcode checks if the opcode is valid
func (f *Frame) isValidOpcode() bool {
	switch f.Opcode {
	case OpcodeContinuation, OpcodeText, OpcodeBinary, OpcodeClose, OpcodePing, OpcodePong:
		return true
	default:
		return false
	}
}

// IsControlFrame returns true if this is a control frame
func (f *Frame) IsControlFrame() bool {
	return f.Opcode.IsControl()
}

// IsDataFrame returns true if this is a data frame
func (f *Frame) IsDataFrame() bool {
	return f.Opcode.IsData()
}
