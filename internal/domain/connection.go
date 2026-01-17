package domain

import (
	"fmt"
	"time"
)

// ConnectionState represents the state of a WebSocket connection
type ConnectionState int

const (
	// StateConnecting indicates the connection is being established
	StateConnecting ConnectionState = iota
	// StateOpen indicates the connection is open and ready
	StateOpen
	// StateClosing indicates the connection is closing
	StateClosing
	// StateClosed indicates the connection is closed
	StateClosed
)

// String returns the string representation of the connection state
func (s ConnectionState) String() string {
	switch s {
	case StateConnecting:
		return "Connecting"
	case StateOpen:
		return "Open"
	case StateClosing:
		return "Closing"
	case StateClosed:
		return "Closed"
	default:
		return fmt.Sprintf("Unknown(%d)", int(s))
	}
}

// Connection represents a WebSocket connection
type Connection struct {
	ID           string                 // Unique connection identifier
	RemoteAddr   string                 // Remote address
	State        ConnectionState        // Current connection state
	LastActivity time.Time              // Last activity timestamp
	Metadata     map[string]interface{} // Connection metadata
}

// NewConnection creates a new connection with the given ID and remote address
func NewConnection(id, remoteAddr string) *Connection {
	return &Connection{
		ID:           id,
		RemoteAddr:   remoteAddr,
		State:        StateConnecting,
		LastActivity: time.Now(),
		Metadata:     make(map[string]interface{}),
	}
}

// CanTransitionTo checks if the connection can transition to the given state
func (c *Connection) CanTransitionTo(newState ConnectionState) bool {
	switch c.State {
	case StateConnecting:
		return newState == StateOpen || newState == StateClosed
	case StateOpen:
		return newState == StateClosing || newState == StateClosed
	case StateClosing:
		return newState == StateClosed
	case StateClosed:
		return false
	default:
		return false
	}
}

// TransitionTo transitions the connection to the given state
func (c *Connection) TransitionTo(newState ConnectionState) error {
	if !c.CanTransitionTo(newState) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidState, c.State, newState)
	}
	c.State = newState
	return nil
}

// UpdateActivity updates the last activity timestamp
func (c *Connection) UpdateActivity() {
	c.LastActivity = time.Now()
}

// IsOpen returns true if the connection is open
func (c *Connection) IsOpen() bool {
	return c.State == StateOpen
}

// IsClosed returns true if the connection is closed
func (c *Connection) IsClosed() bool {
	return c.State == StateClosed
}

// IsClosing returns true if the connection is closing
func (c *Connection) IsClosing() bool {
	return c.State == StateClosing
}
