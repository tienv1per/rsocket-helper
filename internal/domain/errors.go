package domain

import "errors"

// Domain errors
var (
	// Frame errors
	ErrInvalidFrameStructure = errors.New("invalid frame structure")
	ErrInvalidOpcode         = errors.New("invalid opcode")
	ErrReservedBitsSet       = errors.New("reserved bits incorrectly set")
	ErrPayloadTooLarge       = errors.New("payload exceeds maximum size")
	ErrUnmaskedClientFrame   = errors.New("client frame must be masked")
	ErrMaskedServerFrame     = errors.New("server frame must not be masked")

	// Connection errors
	ErrConnectionClosed   = errors.New("connection is closed")
	ErrInvalidState       = errors.New("invalid connection state")
	ErrConnectionNotFound = errors.New("connection not found")

	// Message errors
	ErrInvalidMessageType = errors.New("invalid message type")
	ErrEmptyPayload       = errors.New("empty payload")

	// Protocol errors
	ErrProtocolViolation = errors.New("protocol violation")
	ErrPolicyViolation   = errors.New("policy violation")
	ErrInternalError     = errors.New("internal error")
)
