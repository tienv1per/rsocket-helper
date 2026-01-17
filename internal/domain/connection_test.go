package domain

import (
	"testing"
	"time"
)

func TestNewConnection(t *testing.T) {
	id := "conn-123"
	remoteAddr := "192.168.1.1:8080"

	conn := NewConnection(id, remoteAddr)

	if conn.ID != id {
		t.Errorf("expected ID to be %s, got %s", id, conn.ID)
	}
	if conn.RemoteAddr != remoteAddr {
		t.Errorf("expected RemoteAddr to be %s, got %s", remoteAddr, conn.RemoteAddr)
	}
	if conn.State != StateConnecting {
		t.Errorf("expected State to be Connecting, got %s", conn.State)
	}
	if conn.Metadata == nil {
		t.Error("expected Metadata to be initialized")
	}
	if time.Since(conn.LastActivity) > time.Second {
		t.Error("expected LastActivity to be recent")
	}
}

func TestConnectionStateString(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateConnecting, "Connecting"},
		{StateOpen, "Open"},
		{StateClosing, "Closing"},
		{StateClosed, "Closed"},
		{ConnectionState(99), "Unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConnectionCanTransitionTo(t *testing.T) {
	tests := []struct {
		name      string
		fromState ConnectionState
		toState   ConnectionState
		expected  bool
	}{
		// From Connecting
		{"Connecting to Open", StateConnecting, StateOpen, true},
		{"Connecting to Closed", StateConnecting, StateClosed, true},
		{"Connecting to Closing", StateConnecting, StateClosing, false},
		{"Connecting to Connecting", StateConnecting, StateConnecting, false},

		// From Open
		{"Open to Closing", StateOpen, StateClosing, true},
		{"Open to Closed", StateOpen, StateClosed, true},
		{"Open to Connecting", StateOpen, StateConnecting, false},
		{"Open to Open", StateOpen, StateOpen, false},

		// From Closing
		{"Closing to Closed", StateClosing, StateClosed, true},
		{"Closing to Open", StateClosing, StateOpen, false},
		{"Closing to Connecting", StateClosing, StateConnecting, false},
		{"Closing to Closing", StateClosing, StateClosing, false},

		// From Closed
		{"Closed to any state", StateClosed, StateOpen, false},
		{"Closed to Connecting", StateClosed, StateConnecting, false},
		{"Closed to Closing", StateClosed, StateClosing, false},
		{"Closed to Closed", StateClosed, StateClosed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &Connection{State: tt.fromState}
			if got := conn.CanTransitionTo(tt.toState); got != tt.expected {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConnectionTransitionTo(t *testing.T) {
	tests := []struct {
		name      string
		fromState ConnectionState
		toState   ConnectionState
		wantErr   bool
	}{
		{"valid transition: Connecting to Open", StateConnecting, StateOpen, false},
		{"valid transition: Open to Closing", StateOpen, StateClosing, false},
		{"valid transition: Closing to Closed", StateClosing, StateClosed, false},
		{"invalid transition: Connecting to Closing", StateConnecting, StateClosing, true},
		{"invalid transition: Closed to Open", StateClosed, StateOpen, true},
		{"invalid transition: Open to Connecting", StateOpen, StateConnecting, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &Connection{State: tt.fromState}
			err := conn.TransitionTo(tt.toState)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if conn.State != tt.toState {
					t.Errorf("expected state to be %s, got %s", tt.toState, conn.State)
				}
			}
		})
	}
}

func TestConnectionUpdateActivity(t *testing.T) {
	conn := NewConnection("test", "127.0.0.1:8080")
	oldActivity := conn.LastActivity

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	conn.UpdateActivity()

	if !conn.LastActivity.After(oldActivity) {
		t.Error("expected LastActivity to be updated")
	}
}

func TestConnectionIsOpen(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected bool
	}{
		{StateConnecting, false},
		{StateOpen, true},
		{StateClosing, false},
		{StateClosed, false},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			conn := &Connection{State: tt.state}
			if got := conn.IsOpen(); got != tt.expected {
				t.Errorf("IsOpen() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConnectionIsClosed(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected bool
	}{
		{StateConnecting, false},
		{StateOpen, false},
		{StateClosing, false},
		{StateClosed, true},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			conn := &Connection{State: tt.state}
			if got := conn.IsClosed(); got != tt.expected {
				t.Errorf("IsClosed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConnectionIsClosing(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected bool
	}{
		{StateConnecting, false},
		{StateOpen, false},
		{StateClosing, true},
		{StateClosed, false},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			conn := &Connection{State: tt.state}
			if got := conn.IsClosing(); got != tt.expected {
				t.Errorf("IsClosing() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConnectionStateTransitions(t *testing.T) {
	// Test full lifecycle
	conn := NewConnection("test", "127.0.0.1:8080")

	// Should start in Connecting state
	if conn.State != StateConnecting {
		t.Errorf("expected initial state to be Connecting, got %s", conn.State)
	}

	// Transition to Open
	if err := conn.TransitionTo(StateOpen); err != nil {
		t.Errorf("unexpected error transitioning to Open: %v", err)
	}
	if !conn.IsOpen() {
		t.Error("expected connection to be open")
	}

	// Transition to Closing
	if err := conn.TransitionTo(StateClosing); err != nil {
		t.Errorf("unexpected error transitioning to Closing: %v", err)
	}
	if !conn.IsClosing() {
		t.Error("expected connection to be closing")
	}

	// Transition to Closed
	if err := conn.TransitionTo(StateClosed); err != nil {
		t.Errorf("unexpected error transitioning to Closed: %v", err)
	}
	if !conn.IsClosed() {
		t.Error("expected connection to be closed")
	}

	// Should not be able to transition from Closed
	if err := conn.TransitionTo(StateOpen); err == nil {
		t.Error("expected error when transitioning from Closed state")
	}
}
