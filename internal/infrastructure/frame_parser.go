package infrastructure

import (
	"encoding/binary"
	"io"

	"websocket-server/internal/domain"
	"websocket-server/pkg/protocol"
)

// FrameParser handles parsing and construction of WebSocket frames
type FrameParser struct {
	maxPayloadSize uint64
}

// NewFrameParser creates a new frame parser with the given maximum payload size
func NewFrameParser(maxPayloadSize uint64) *FrameParser {
	if maxPayloadSize == 0 {
		maxPayloadSize = protocol.MaxPayloadSize
	}
	return &FrameParser{
		maxPayloadSize: maxPayloadSize,
	}
}

// ReadFrame reads and parses a WebSocket frame from the reader
func (fp *FrameParser) ReadFrame(reader io.Reader) (*domain.Frame, error) {
	frame := &domain.Frame{}

	// Read first two bytes (minimum frame header)
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}

	// Parse first byte: FIN, RSV1-3, Opcode
	frame.FIN = (header[0] & 0x80) != 0
	frame.RSV1 = (header[0] & 0x40) != 0
	frame.RSV2 = (header[0] & 0x20) != 0
	frame.RSV3 = (header[0] & 0x10) != 0
	frame.Opcode = domain.Opcode(header[0] & 0x0F)

	// Parse second byte: MASK, Payload length
	frame.Masked = (header[1] & 0x80) != 0
	payloadLen := uint64(header[1] & 0x7F)

	// Validate frame structure
	// Check if opcode is valid
	if !frame.Opcode.IsControl() && !frame.Opcode.IsData() {
		return nil, domain.ErrInvalidOpcode
	}

	// Check if reserved bits are set (they should be 0 unless extensions are negotiated)
	if frame.RSV1 || frame.RSV2 || frame.RSV3 {
		return nil, domain.ErrReservedBitsSet
	}

	// Parse extended payload length if needed
	var err error
	payloadLen, err = fp.parsePayloadLength(reader, payloadLen)
	if err != nil {
		return nil, err
	}

	frame.PayloadLen = payloadLen

	// Check payload size limit
	if payloadLen > fp.maxPayloadSize {
		return nil, domain.ErrPayloadTooLarge
	}

	// Control frames must have payload length <= 125
	if frame.Opcode.IsControl() && payloadLen > 125 {
		return nil, domain.ErrInvalidFrameStructure
	}

	// Control frames must not be fragmented
	if frame.Opcode.IsControl() && !frame.FIN {
		return nil, domain.ErrInvalidFrameStructure
	}

	// Read masking key if present
	if frame.Masked {
		if _, err := io.ReadFull(reader, frame.MaskingKey[:]); err != nil {
			return nil, err
		}
	}

	// Read payload
	if payloadLen > 0 {
		frame.Payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(reader, frame.Payload); err != nil {
			return nil, err
		}

		// Unmask payload if masked
		if frame.Masked {
			fp.UnmaskPayload(frame.Payload, frame.MaskingKey)
		}
	}

	return frame, nil
}

// parsePayloadLength parses the payload length based on the initial length value
func (fp *FrameParser) parsePayloadLength(reader io.Reader, initialLen uint64) (uint64, error) {
	switch initialLen {
	case protocol.PayloadLen16Bit:
		// 16-bit extended payload length
		buf := make([]byte, 2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return 0, err
		}
		return uint64(binary.BigEndian.Uint16(buf)), nil

	case protocol.PayloadLen64Bit:
		// 64-bit extended payload length
		buf := make([]byte, 8)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return 0, err
		}
		return binary.BigEndian.Uint64(buf), nil

	default:
		// 7-bit payload length
		return initialLen, nil
	}
}

// UnmaskPayload unmasks the payload using the masking key
func (fp *FrameParser) UnmaskPayload(payload []byte, maskingKey [4]byte) {
	for i := range payload {
		payload[i] ^= maskingKey[i%4]
	}
}

// WriteFrame writes a WebSocket frame to the writer
func (fp *FrameParser) WriteFrame(writer io.Writer, frame *domain.Frame) error {
	// Validate frame before writing
	if err := frame.Validate(); err != nil {
		return err
	}

	// Build frame header
	header := make([]byte, 0, 14) // Max header size

	// First byte: FIN, RSV1-3, Opcode
	firstByte := byte(frame.Opcode)
	if frame.FIN {
		firstByte |= 0x80
	}
	if frame.RSV1 {
		firstByte |= 0x40
	}
	if frame.RSV2 {
		firstByte |= 0x20
	}
	if frame.RSV3 {
		firstByte |= 0x10
	}
	header = append(header, firstByte)

	// Second byte: MASK, Payload length
	payloadLen := frame.PayloadLen
	secondByte := byte(0)
	if frame.Masked {
		secondByte |= 0x80
	}

	// Determine payload length encoding
	if payloadLen <= 125 {
		secondByte |= byte(payloadLen)
		header = append(header, secondByte)
	} else if payloadLen <= 65535 {
		secondByte |= protocol.PayloadLen16Bit
		header = append(header, secondByte)
		// Add 16-bit extended length
		extLen := make([]byte, 2)
		binary.BigEndian.PutUint16(extLen, uint16(payloadLen))
		header = append(header, extLen...)
	} else {
		secondByte |= protocol.PayloadLen64Bit
		header = append(header, secondByte)
		// Add 64-bit extended length
		extLen := make([]byte, 8)
		binary.BigEndian.PutUint64(extLen, payloadLen)
		header = append(header, extLen...)
	}

	// Add masking key if masked
	if frame.Masked {
		header = append(header, frame.MaskingKey[:]...)
	}

	// Write header
	if _, err := writer.Write(header); err != nil {
		return err
	}

	// Write payload (mask if needed)
	if len(frame.Payload) > 0 {
		payload := frame.Payload
		if frame.Masked {
			// Create a copy to avoid modifying the original
			payload = make([]byte, len(frame.Payload))
			copy(payload, frame.Payload)
			fp.UnmaskPayload(payload, frame.MaskingKey)
		}
		if _, err := writer.Write(payload); err != nil {
			return err
		}
	}

	return nil
}
