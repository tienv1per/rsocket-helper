package infrastructure

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"websocket-server/pkg/protocol"
)

// HandshakeValidator validates WebSocket handshake requests and performs upgrades
type HandshakeValidator struct{}

// NewHandshakeValidator creates a new HandshakeValidator
func NewHandshakeValidator() *HandshakeValidator {
	return &HandshakeValidator{}
}

// ValidateRequest validates that the HTTP request contains all required WebSocket handshake headers
func (h *HandshakeValidator) ValidateRequest(req *http.Request) error {
	// Validate Upgrade header
	upgrade := req.Header.Get(protocol.HeaderUpgrade)
	if !strings.EqualFold(upgrade, protocol.HeaderValueWebSocket) {
		return fmt.Errorf("missing or invalid Upgrade header: expected 'websocket', got '%s'", upgrade)
	}

	// Validate Connection header
	connection := req.Header.Get(protocol.HeaderConnection)
	if !containsToken(connection, protocol.HeaderValueUpgrade) {
		return fmt.Errorf("missing or invalid Connection header: expected 'Upgrade', got '%s'", connection)
	}

	// Validate Sec-WebSocket-Key header
	key := req.Header.Get(protocol.HeaderSecWebSocketKey)
	if key == "" {
		return fmt.Errorf("missing Sec-WebSocket-Key header")
	}

	// Validate Sec-WebSocket-Version header
	version := req.Header.Get(protocol.HeaderSecWebSocketVersion)
	if version != protocol.WebSocketVersion {
		return fmt.Errorf("unsupported WebSocket version: expected '%s', got '%s'", protocol.WebSocketVersion, version)
	}

	return nil
}

// GenerateAcceptKey generates the Sec-WebSocket-Accept value from the client's key
// According to RFC 6455: base64(SHA1(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
func (h *HandshakeValidator) GenerateAcceptKey(key string) string {
	// Concatenate key with magic GUID
	combined := key + protocol.WebSocketGUID

	// Compute SHA-1 hash
	hash := sha1.Sum([]byte(combined))

	// Encode as base64
	return base64.StdEncoding.EncodeToString(hash[:])
}

// PerformUpgrade performs the WebSocket upgrade handshake
func (h *HandshakeValidator) PerformUpgrade(w http.ResponseWriter, req *http.Request) error {
	// Validate the request
	if err := h.ValidateRequest(req); err != nil {
		// Send HTTP 400 Bad Request for invalid handshakes
		http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return err
	}

	// Get the Sec-WebSocket-Key
	key := req.Header.Get(protocol.HeaderSecWebSocketKey)

	// Generate the accept key
	acceptKey := h.GenerateAcceptKey(key)

	// Send HTTP 101 Switching Protocols response
	w.Header().Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
	w.Header().Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
	w.Header().Set(protocol.HeaderSecWebSocketAccept, acceptKey)
	w.WriteHeader(http.StatusSwitchingProtocols)

	return nil
}

// containsToken checks if a comma-separated header value contains a specific token (case-insensitive)
func containsToken(header, token string) bool {
	tokens := strings.Split(header, ",")
	for _, t := range tokens {
		if strings.EqualFold(strings.TrimSpace(t), token) {
			return true
		}
	}
	return false
}
