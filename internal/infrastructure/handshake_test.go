package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"websocket-server/pkg/protocol"
)

// Feature: websocket-server, Property 2: Handshake Validation Completeness
// Validates: Requirements 2.2, 2.3, 2.4, 2.5, 8.3
func TestProperty_HandshakeValidationCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	validator := NewHandshakeValidator()

	// Property: For any HTTP request missing any required header, validation SHALL reject it
	properties.Property("missing Upgrade header should be rejected", prop.ForAll(
		func(key, version string) bool {
			req := httptest.NewRequest("GET", "/", nil)
			// Missing Upgrade header
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, version)

			err := validator.ValidateRequest(req)
			return err != nil // Should fail validation
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("missing Connection header should be rejected", prop.ForAll(
		func(key, version string) bool {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
			// Missing Connection header
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, version)

			err := validator.ValidateRequest(req)
			return err != nil // Should fail validation
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("missing Sec-WebSocket-Key header should be rejected", prop.ForAll(
		func(version string) bool {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			// Missing Sec-WebSocket-Key header
			req.Header.Set(protocol.HeaderSecWebSocketVersion, version)

			err := validator.ValidateRequest(req)
			return err != nil // Should fail validation
		},
		gen.Identifier(),
	))

	properties.Property("missing Sec-WebSocket-Version header should be rejected", prop.ForAll(
		func(key string) bool {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			// Missing Sec-WebSocket-Version header

			err := validator.ValidateRequest(req)
			return err != nil // Should fail validation
		},
		gen.Identifier(),
	))

	properties.Property("invalid Upgrade header value should be rejected", prop.ForAll(
		func(key, invalidUpgrade string) bool {
			// Skip if randomly generated value happens to be "websocket"
			if invalidUpgrade == protocol.HeaderValueWebSocket || invalidUpgrade == "" {
				return true
			}

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, invalidUpgrade)
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, protocol.WebSocketVersion)

			err := validator.ValidateRequest(req)
			return err != nil // Should fail validation
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("invalid Connection header value should be rejected", prop.ForAll(
		func(key, invalidConnection string) bool {
			// Skip if randomly generated value happens to contain "Upgrade"
			if invalidConnection == protocol.HeaderValueUpgrade || invalidConnection == "" {
				return true
			}

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
			req.Header.Set(protocol.HeaderConnection, invalidConnection)
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, protocol.WebSocketVersion)

			err := validator.ValidateRequest(req)
			return err != nil // Should fail validation
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("invalid Sec-WebSocket-Version should be rejected", prop.ForAll(
		func(key, invalidVersion string) bool {
			// Skip if randomly generated value happens to be "13"
			if invalidVersion == protocol.WebSocketVersion || invalidVersion == "" {
				return true
			}

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, invalidVersion)

			err := validator.ValidateRequest(req)
			return err != nil // Should fail validation
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("valid handshake with all required headers should be accepted", prop.ForAll(
		func(key string) bool {
			// Skip empty keys
			if key == "" {
				return true
			}

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, protocol.WebSocketVersion)

			err := validator.ValidateRequest(req)
			return err == nil // Should pass validation
		},
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// Feature: websocket-server, Property 1: Handshake Accept Key Computation
// Validates: Requirements 2.6
func TestProperty_HandshakeAcceptKeyComputation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	validator := NewHandshakeValidator()

	// Property: For any valid Sec-WebSocket-Key, computing the accept key SHALL produce
	// the correct result according to RFC 6455: base64(SHA1(key + GUID))
	properties.Property("accept key computation follows RFC 6455", prop.ForAll(
		func(key string) bool {
			// Skip empty keys
			if key == "" {
				return true
			}

			acceptKey := validator.GenerateAcceptKey(key)

			// Verify the accept key is not empty
			if acceptKey == "" {
				return false
			}

			// Verify the accept key is base64 encoded (should be 28 characters for SHA-1)
			if len(acceptKey) != 28 {
				return false
			}

			// Verify idempotence: calling it again with the same key produces the same result
			acceptKey2 := validator.GenerateAcceptKey(key)
			if acceptKey != acceptKey2 {
				return false
			}

			return true
		},
		gen.Identifier(),
	))

	// Test with known values from RFC 6455
	properties.Property("RFC 6455 example key produces correct accept key", prop.ForAll(
		func() bool {
			// Example from RFC 6455 Section 1.3
			key := "dGhlIHNhbXBsZSBub25jZQ=="
			expectedAccept := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="

			acceptKey := validator.GenerateAcceptKey(key)

			return acceptKey == expectedAccept
		},
	))

	// Property: Different keys should produce different accept keys
	properties.Property("different keys produce different accept keys", prop.ForAll(
		func(key1, key2 string) bool {
			// Skip if keys are the same or empty
			if key1 == key2 || key1 == "" || key2 == "" {
				return true
			}

			acceptKey1 := validator.GenerateAcceptKey(key1)
			acceptKey2 := validator.GenerateAcceptKey(key2)

			return acceptKey1 != acceptKey2
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// Feature: websocket-server, Property 3: Valid Handshake Response
// Validates: Requirements 2.7
func TestProperty_ValidHandshakeResponse(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	validator := NewHandshakeValidator()

	// Property: For any valid WebSocket handshake request, the server SHALL respond
	// with HTTP 101 Switching Protocols status
	properties.Property("valid handshake returns 101 status", prop.ForAll(
		func(key string) bool {
			// Skip empty keys
			if key == "" {
				return true
			}

			// Create a valid handshake request
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, protocol.WebSocketVersion)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Perform the upgrade
			err := validator.PerformUpgrade(w, req)

			// Should not return an error
			if err != nil {
				return false
			}

			// Should return 101 Switching Protocols
			if w.Code != http.StatusSwitchingProtocols {
				return false
			}

			// Should have correct response headers
			if w.Header().Get(protocol.HeaderUpgrade) != protocol.HeaderValueWebSocket {
				return false
			}
			if w.Header().Get(protocol.HeaderConnection) != protocol.HeaderValueUpgrade {
				return false
			}

			// Should have Sec-WebSocket-Accept header
			acceptKey := w.Header().Get(protocol.HeaderSecWebSocketAccept)
			if acceptKey == "" {
				return false
			}

			// Accept key should match the expected value
			expectedAcceptKey := validator.GenerateAcceptKey(key)
			if acceptKey != expectedAcceptKey {
				return false
			}

			return true
		},
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// Feature: websocket-server, Property 4: Invalid Handshake Response
// Validates: Requirements 2.8
func TestProperty_InvalidHandshakeResponse(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	validator := NewHandshakeValidator()

	// Property: For any invalid WebSocket handshake request, the server SHALL respond
	// with HTTP 400 Bad Request status
	properties.Property("missing Upgrade header returns 400", prop.ForAll(
		func(key string) bool {
			// Skip empty keys
			if key == "" {
				return true
			}

			// Create an invalid handshake request (missing Upgrade header)
			req := httptest.NewRequest("GET", "/", nil)
			// Missing Upgrade header
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, protocol.WebSocketVersion)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Perform the upgrade
			err := validator.PerformUpgrade(w, req)

			// Should return an error
			if err == nil {
				return false
			}

			// Should return 400 Bad Request
			if w.Code != http.StatusBadRequest {
				return false
			}

			return true
		},
		gen.Identifier(),
	))

	properties.Property("missing Connection header returns 400", prop.ForAll(
		func(key string) bool {
			// Skip empty keys
			if key == "" {
				return true
			}

			// Create an invalid handshake request (missing Connection header)
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
			// Missing Connection header
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, protocol.WebSocketVersion)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Perform the upgrade
			err := validator.PerformUpgrade(w, req)

			// Should return an error
			if err == nil {
				return false
			}

			// Should return 400 Bad Request
			if w.Code != http.StatusBadRequest {
				return false
			}

			return true
		},
		gen.Identifier(),
	))

	properties.Property("missing Sec-WebSocket-Key returns 400", prop.ForAll(
		func() bool {
			// Create an invalid handshake request (missing Sec-WebSocket-Key)
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			// Missing Sec-WebSocket-Key
			req.Header.Set(protocol.HeaderSecWebSocketVersion, protocol.WebSocketVersion)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Perform the upgrade
			err := validator.PerformUpgrade(w, req)

			// Should return an error
			if err == nil {
				return false
			}

			// Should return 400 Bad Request
			if w.Code != http.StatusBadRequest {
				return false
			}

			return true
		},
	))

	properties.Property("invalid Sec-WebSocket-Version returns 400", prop.ForAll(
		func(key, invalidVersion string) bool {
			// Skip empty keys or if version happens to be valid
			if key == "" || invalidVersion == protocol.WebSocketVersion {
				return true
			}

			// Create an invalid handshake request (invalid version)
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, protocol.HeaderValueWebSocket)
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, invalidVersion)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Perform the upgrade
			err := validator.PerformUpgrade(w, req)

			// Should return an error
			if err == nil {
				return false
			}

			// Should return 400 Bad Request
			if w.Code != http.StatusBadRequest {
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("invalid Upgrade header value returns 400", prop.ForAll(
		func(key, invalidUpgrade string) bool {
			// Skip empty keys or if upgrade happens to be valid
			if key == "" || invalidUpgrade == protocol.HeaderValueWebSocket {
				return true
			}

			// Create an invalid handshake request (invalid Upgrade value)
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(protocol.HeaderUpgrade, invalidUpgrade)
			req.Header.Set(protocol.HeaderConnection, protocol.HeaderValueUpgrade)
			req.Header.Set(protocol.HeaderSecWebSocketKey, key)
			req.Header.Set(protocol.HeaderSecWebSocketVersion, protocol.WebSocketVersion)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Perform the upgrade
			err := validator.PerformUpgrade(w, req)

			// Should return an error
			if err == nil {
				return false
			}

			// Should return 400 Bad Request
			if w.Code != http.StatusBadRequest {
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t)
}
