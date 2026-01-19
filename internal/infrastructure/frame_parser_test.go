package infrastructure

import (
	"bytes"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"websocket-server/internal/domain"
	"websocket-server/pkg/protocol"
)

// Feature: websocket-server, Property 5: Frame Header Parsing Correctness
// Validates: Requirements 3.1
func TestProperty_FrameHeaderParsingCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("parsing frame header extracts all fields correctly", prop.ForAll(
		func(fin bool, opcodeIdx int, masked bool, payloadLen uint64) bool {
			// Only test valid opcodes
			validOpcodes := []domain.Opcode{
				domain.OpcodeContinuation,
				domain.OpcodeText,
				domain.OpcodeBinary,
				domain.OpcodeClose,
				domain.OpcodePing,
				domain.OpcodePong,
			}
			frameOpcode := validOpcodes[opcodeIdx]

			// Control frames must not be fragmented (FIN must be 1)
			if frameOpcode.IsControl() && !fin {
				fin = true
			}

			// Control frames must have payload <= 125
			if frameOpcode.IsControl() && payloadLen > 125 {
				payloadLen = 125
			}

			// Limit payload size for testing
			if payloadLen > 1000 {
				payloadLen = 1000
			}

			// Reserved bits must be false for valid frames (no extensions negotiated)
			rsv1, rsv2, rsv3 := false, false, false

			// Build frame bytes manually
			var buf bytes.Buffer

			// First byte: FIN, RSV1-3, Opcode
			firstByte := byte(frameOpcode)
			if fin {
				firstByte |= 0x80
			}
			if rsv1 {
				firstByte |= 0x40
			}
			if rsv2 {
				firstByte |= 0x20
			}
			if rsv3 {
				firstByte |= 0x10
			}
			buf.WriteByte(firstByte)

			// Second byte: MASK, Payload length
			secondByte := byte(0)
			if masked {
				secondByte |= 0x80
			}

			// Write payload length
			if payloadLen <= 125 {
				secondByte |= byte(payloadLen)
				buf.WriteByte(secondByte)
			} else if payloadLen <= 65535 {
				secondByte |= protocol.PayloadLen16Bit
				buf.WriteByte(secondByte)
				buf.WriteByte(byte(payloadLen >> 8))
				buf.WriteByte(byte(payloadLen))
			} else {
				secondByte |= protocol.PayloadLen64Bit
				buf.WriteByte(secondByte)
				for i := 7; i >= 0; i-- {
					buf.WriteByte(byte(payloadLen >> (i * 8)))
				}
			}

			// Write masking key if masked
			maskingKey := [4]byte{0x12, 0x34, 0x56, 0x78}
			if masked {
				buf.Write(maskingKey[:])
			}

			// Write payload
			payload := make([]byte, payloadLen)
			for i := range payload {
				payload[i] = byte(i % 256)
			}
			if masked {
				// Mask the payload before writing
				maskedPayload := make([]byte, len(payload))
				copy(maskedPayload, payload)
				for i := range maskedPayload {
					maskedPayload[i] ^= maskingKey[i%4]
				}
				buf.Write(maskedPayload)
			} else {
				buf.Write(payload)
			}

			// Parse the frame
			parser := NewFrameParser(protocol.MaxPayloadSize)
			frame, err := parser.ReadFrame(&buf)
			if err != nil {
				t.Logf("Error parsing frame: %v", err)
				return false
			}

			// Verify all header fields
			if frame.FIN != fin {
				t.Logf("FIN mismatch: expected %v, got %v", fin, frame.FIN)
				return false
			}
			if frame.RSV1 != rsv1 {
				t.Logf("RSV1 mismatch: expected %v, got %v", rsv1, frame.RSV1)
				return false
			}
			if frame.RSV2 != rsv2 {
				t.Logf("RSV2 mismatch: expected %v, got %v", rsv2, frame.RSV2)
				return false
			}
			if frame.RSV3 != rsv3 {
				t.Logf("RSV3 mismatch: expected %v, got %v", rsv3, frame.RSV3)
				return false
			}
			if frame.Opcode != frameOpcode {
				t.Logf("Opcode mismatch: expected %v, got %v", frameOpcode, frame.Opcode)
				return false
			}
			if frame.Masked != masked {
				t.Logf("Masked mismatch: expected %v, got %v", masked, frame.Masked)
				return false
			}
			if frame.PayloadLen != payloadLen {
				t.Logf("PayloadLen mismatch: expected %v, got %v", payloadLen, frame.PayloadLen)
				return false
			}

			// Verify payload
			if !bytes.Equal(frame.Payload, payload) {
				t.Logf("Payload mismatch")
				return false
			}

			return true
		},
		gen.Bool(),                 // fin
		gen.IntRange(0, 5),         // opcodeIdx
		gen.Bool(),                 // masked
		gen.UInt64Range(0, 100000), // payloadLen
	))

	properties.TestingRun(t)
}

// Feature: websocket-server, Property 7: Payload Unmasking Correctness
// Validates: Requirements 3.7
func TestProperty_PayloadUnmaskingCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("unmasking payload correctly recovers original data", prop.ForAll(
		func(payloadLen int, maskKey1 byte, maskKey2 byte, maskKey3 byte, maskKey4 byte) bool {
			// Limit payload size
			if payloadLen > 1000 {
				payloadLen = 1000
			}
			if payloadLen < 0 {
				payloadLen = 0
			}

			maskingKey := [4]byte{maskKey1, maskKey2, maskKey3, maskKey4}

			// Create original payload
			payload := make([]byte, payloadLen)
			for i := range payload {
				payload[i] = byte(i % 256)
			}

			// Create a copy of the original payload
			original := make([]byte, len(payload))
			copy(original, payload)

			// Mask the payload
			parser := NewFrameParser(protocol.MaxPayloadSize)
			parser.UnmaskPayload(payload, maskingKey)

			// Unmask it again (XOR is its own inverse)
			parser.UnmaskPayload(payload, maskingKey)

			// Should get back the original
			if !bytes.Equal(payload, original) {
				t.Logf("Unmasking failed: original != unmasked")
				return false
			}

			return true
		},
		gen.IntRange(0, 1000), // payloadLen
		gen.UInt8(),           // maskKey1
		gen.UInt8(),           // maskKey2
		gen.UInt8(),           // maskKey3
		gen.UInt8(),           // maskKey4
	))

	properties.TestingRun(t)
}

// Feature: websocket-server, Property 8: Server Frame Masking
// Validates: Requirements 3.8
func TestProperty_ServerFrameMasking(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("server frames must not be masked", prop.ForAll(
		func(opcodeIdx int, payloadLen int) bool {
			// Only test valid opcodes
			validOpcodes := []domain.Opcode{
				domain.OpcodeContinuation,
				domain.OpcodeText,
				domain.OpcodeBinary,
				domain.OpcodeClose,
				domain.OpcodePing,
				domain.OpcodePong,
			}
			frameOpcode := validOpcodes[opcodeIdx]

			// Control frames must have payload <= 125
			if frameOpcode.IsControl() && payloadLen > 125 {
				payloadLen = 125
			}

			// Limit payload size for testing
			if payloadLen > 1000 {
				payloadLen = 1000
			}
			if payloadLen < 0 {
				payloadLen = 0
			}

			// Create payload
			payload := make([]byte, payloadLen)
			for i := range payload {
				payload[i] = byte(i % 256)
			}

			// Create frame (server frames should not be masked)
			frame := domain.NewFrame(frameOpcode, payload)

			// Write frame
			var buf bytes.Buffer
			parser := NewFrameParser(protocol.MaxPayloadSize)
			if err := parser.WriteFrame(&buf, frame); err != nil {
				t.Logf("Error writing frame: %v", err)
				return false
			}

			// Read back the frame header to check mask bit
			frameBytes := buf.Bytes()
			if len(frameBytes) < 2 {
				t.Logf("Frame too short")
				return false
			}

			// Check mask bit (bit 7 of second byte)
			maskBit := (frameBytes[1] & 0x80) != 0
			if maskBit {
				t.Logf("Server frame should not be masked, but mask bit is set")
				return false
			}

			return true
		},
		gen.IntRange(0, 5),    // opcodeIdx
		gen.IntRange(0, 1000), // payloadLen
	))

	properties.TestingRun(t)
}

// Feature: websocket-server, Property 9: Frame Opcode Correctness
// Validates: Requirements 3.9, 4.1, 4.2
func TestProperty_FrameOpcodeCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("frame opcode matches message type", prop.ForAll(
		func(isText bool, payloadLen int) bool {
			// Limit payload size for testing
			if payloadLen > 1000 {
				payloadLen = 1000
			}
			if payloadLen < 0 {
				payloadLen = 0
			}

			// Create payload
			payload := make([]byte, payloadLen)
			for i := range payload {
				payload[i] = byte(i % 256)
			}

			// Determine opcode based on message type
			var expectedOpcode domain.Opcode
			if isText {
				expectedOpcode = domain.OpcodeText
			} else {
				expectedOpcode = domain.OpcodeBinary
			}

			// Create frame
			frame := domain.NewFrame(expectedOpcode, payload)

			// Write and read back
			var buf bytes.Buffer
			parser := NewFrameParser(protocol.MaxPayloadSize)
			if err := parser.WriteFrame(&buf, frame); err != nil {
				t.Logf("Error writing frame: %v", err)
				return false
			}

			// Parse the frame
			parsedFrame, err := parser.ReadFrame(&buf)
			if err != nil {
				t.Logf("Error parsing frame: %v", err)
				return false
			}

			// Verify opcode
			if parsedFrame.Opcode != expectedOpcode {
				t.Logf("Opcode mismatch: expected %v, got %v", expectedOpcode, parsedFrame.Opcode)
				return false
			}

			return true
		},
		gen.Bool(),            // isText
		gen.IntRange(0, 1000), // payloadLen
	))

	properties.TestingRun(t)
}

// Feature: websocket-server, Property 10: Invalid Frame Rejection
// Validates: Requirements 3.10, 8.1
func TestProperty_InvalidFrameRejection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("invalid frames are rejected", prop.ForAll(
		func(invalidType int) bool {
			var buf bytes.Buffer
			parser := NewFrameParser(protocol.MaxPayloadSize)

			switch invalidType % 3 {
			case 0:
				// Invalid opcode (0x3 is reserved)
				buf.WriteByte(0x83) // FIN=1, opcode=0x3
				buf.WriteByte(0x00) // No mask, payload len=0

			case 1:
				// Reserved bit set
				buf.WriteByte(0xC1) // FIN=1, RSV1=1, opcode=0x1 (text)
				buf.WriteByte(0x00) // No mask, payload len=0

			case 2:
				// Control frame with FIN=0 (fragmented control frame)
				buf.WriteByte(0x08) // FIN=0, opcode=0x8 (close)
				buf.WriteByte(0x00) // No mask, payload len=0
			}

			// Try to parse - should fail
			_, err := parser.ReadFrame(&buf)
			if err == nil {
				t.Logf("Expected error for invalid frame type %d, but got none", invalidType%3)
				return false
			}

			// Verify it's one of the expected errors
			if err != domain.ErrInvalidOpcode &&
				err != domain.ErrReservedBitsSet &&
				err != domain.ErrInvalidFrameStructure {
				t.Logf("Unexpected error type: %v", err)
				return false
			}

			return true
		},
		gen.IntRange(0, 100), // invalidType
	))

	properties.TestingRun(t)
}

// Feature: websocket-server, Property 24: Maximum Payload Size Enforcement
// Validates: Requirements 8.2
func TestProperty_MaximumPayloadSizeEnforcement(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("frames exceeding max payload size are rejected", prop.ForAll(
		func(excessSize int) bool {
			// Set a small max payload size for testing
			maxSize := uint64(1000)
			parser := NewFrameParser(maxSize)

			// Create a frame that exceeds the max size
			payloadLen := maxSize + uint64(excessSize%1000) + 1

			// Build frame bytes manually
			var buf bytes.Buffer

			// First byte: FIN=1, opcode=0x1 (text)
			buf.WriteByte(0x81)

			// Second byte and payload length
			if payloadLen <= 125 {
				buf.WriteByte(byte(payloadLen))
			} else if payloadLen <= 65535 {
				buf.WriteByte(126)
				buf.WriteByte(byte(payloadLen >> 8))
				buf.WriteByte(byte(payloadLen))
			} else {
				buf.WriteByte(127)
				for i := 7; i >= 0; i-- {
					buf.WriteByte(byte(payloadLen >> (i * 8)))
				}
			}

			// Try to parse - should fail with ErrPayloadTooLarge
			_, err := parser.ReadFrame(&buf)
			if err != domain.ErrPayloadTooLarge {
				t.Logf("Expected ErrPayloadTooLarge, got: %v", err)
				return false
			}

			return true
		},
		gen.IntRange(0, 1000), // excessSize
	))

	properties.TestingRun(t)
}

// Unit tests for frame type support
// Requirements: 3.2, 3.3, 3.4, 3.5, 3.6

func TestFrameParser_TextFrame(t *testing.T) {
	parser := NewFrameParser(protocol.MaxPayloadSize)
	payload := []byte("Hello, WebSocket!")

	// Create and write text frame
	frame := domain.NewFrame(domain.OpcodeText, payload)
	var buf bytes.Buffer
	if err := parser.WriteFrame(&buf, frame); err != nil {
		t.Fatalf("Failed to write text frame: %v", err)
	}

	// Read and verify
	parsedFrame, err := parser.ReadFrame(&buf)
	if err != nil {
		t.Fatalf("Failed to read text frame: %v", err)
	}

	if parsedFrame.Opcode != domain.OpcodeText {
		t.Errorf("Expected opcode Text, got %v", parsedFrame.Opcode)
	}
	if !bytes.Equal(parsedFrame.Payload, payload) {
		t.Errorf("Payload mismatch")
	}
}

func TestFrameParser_BinaryFrame(t *testing.T) {
	parser := NewFrameParser(protocol.MaxPayloadSize)
	payload := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}

	// Create and write binary frame
	frame := domain.NewFrame(domain.OpcodeBinary, payload)
	var buf bytes.Buffer
	if err := parser.WriteFrame(&buf, frame); err != nil {
		t.Fatalf("Failed to write binary frame: %v", err)
	}

	// Read and verify
	parsedFrame, err := parser.ReadFrame(&buf)
	if err != nil {
		t.Fatalf("Failed to read binary frame: %v", err)
	}

	if parsedFrame.Opcode != domain.OpcodeBinary {
		t.Errorf("Expected opcode Binary, got %v", parsedFrame.Opcode)
	}
	if !bytes.Equal(parsedFrame.Payload, payload) {
		t.Errorf("Payload mismatch")
	}
}

func TestFrameParser_CloseFrame(t *testing.T) {
	parser := NewFrameParser(protocol.MaxPayloadSize)
	payload := []byte{0x03, 0xE8} // Status code 1000

	// Create and write close frame
	frame := domain.NewFrame(domain.OpcodeClose, payload)
	var buf bytes.Buffer
	if err := parser.WriteFrame(&buf, frame); err != nil {
		t.Fatalf("Failed to write close frame: %v", err)
	}

	// Read and verify
	parsedFrame, err := parser.ReadFrame(&buf)
	if err != nil {
		t.Fatalf("Failed to read close frame: %v", err)
	}

	if parsedFrame.Opcode != domain.OpcodeClose {
		t.Errorf("Expected opcode Close, got %v", parsedFrame.Opcode)
	}
	if !bytes.Equal(parsedFrame.Payload, payload) {
		t.Errorf("Payload mismatch")
	}
}

func TestFrameParser_PingFrame(t *testing.T) {
	parser := NewFrameParser(protocol.MaxPayloadSize)
	payload := []byte("ping")

	// Create and write ping frame
	frame := domain.NewFrame(domain.OpcodePing, payload)
	var buf bytes.Buffer
	if err := parser.WriteFrame(&buf, frame); err != nil {
		t.Fatalf("Failed to write ping frame: %v", err)
	}

	// Read and verify
	parsedFrame, err := parser.ReadFrame(&buf)
	if err != nil {
		t.Fatalf("Failed to read ping frame: %v", err)
	}

	if parsedFrame.Opcode != domain.OpcodePing {
		t.Errorf("Expected opcode Ping, got %v", parsedFrame.Opcode)
	}
	if !bytes.Equal(parsedFrame.Payload, payload) {
		t.Errorf("Payload mismatch")
	}
}

func TestFrameParser_PongFrame(t *testing.T) {
	parser := NewFrameParser(protocol.MaxPayloadSize)
	payload := []byte("pong")

	// Create and write pong frame
	frame := domain.NewFrame(domain.OpcodePong, payload)
	var buf bytes.Buffer
	if err := parser.WriteFrame(&buf, frame); err != nil {
		t.Fatalf("Failed to write pong frame: %v", err)
	}

	// Read and verify
	parsedFrame, err := parser.ReadFrame(&buf)
	if err != nil {
		t.Fatalf("Failed to read pong frame: %v", err)
	}

	if parsedFrame.Opcode != domain.OpcodePong {
		t.Errorf("Expected opcode Pong, got %v", parsedFrame.Opcode)
	}
	if !bytes.Equal(parsedFrame.Payload, payload) {
		t.Errorf("Payload mismatch")
	}
}
