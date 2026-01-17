package protocol

// WebSocket protocol constants as defined in RFC 6455

const (
	// WebSocketGUID is the magic string used in handshake accept key calculation
	WebSocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	// WebSocket version
	WebSocketVersion = "13"

	// Header names
	HeaderUpgrade              = "Upgrade"
	HeaderConnection           = "Connection"
	HeaderSecWebSocketKey      = "Sec-WebSocket-Key"
	HeaderSecWebSocketAccept   = "Sec-WebSocket-Accept"
	HeaderSecWebSocketVersion  = "Sec-WebSocket-Version"
	HeaderSecWebSocketProtocol = "Sec-WebSocket-Protocol"

	// Header values
	HeaderValueWebSocket = "websocket"
	HeaderValueUpgrade   = "Upgrade"

	// Close status codes
	StatusNormalClosure           = 1000
	StatusGoingAway               = 1001
	StatusProtocolError           = 1002
	StatusUnsupportedData         = 1003
	StatusNoStatusReceived        = 1005
	StatusAbnormalClosure         = 1006
	StatusInvalidFramePayloadData = 1007
	StatusPolicyViolation         = 1008
	StatusMessageTooBig           = 1009
	StatusMandatoryExtension      = 1010
	StatusInternalServerError     = 1011
	StatusServiceRestart          = 1012
	StatusTryAgainLater           = 1013
	StatusBadGateway              = 1014
	StatusTLSHandshake            = 1015

	// Frame size limits
	MaxControlFramePayloadSize = 125
	MaxPayloadSize             = 1 << 20 // 1MB default max payload size

	// Payload length indicators
	PayloadLen16Bit = 126
	PayloadLen64Bit = 127
)
