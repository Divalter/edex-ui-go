package terminal_test

import (
	"edex-ui-go/internal/config"
	"edex-ui-go/internal/terminal"
	"testing"
)

func TestTerminalManagerLifecycle(t *testing.T) {
	cfg := config.DefaultConfig()
	mgr := terminal.NewManager(cfg)
	if mgr == nil {
		t.Fatal("Expected non-nil manager")
	}

	session, err := mgr.CreateSession("test-tab-1")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be non-nil")
	}

	// Retrieve session
	retrieved, err := mgr.GetSession("test-tab-1")
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if retrieved.ID != "test-tab-1" {
		t.Errorf("Expected session ID 'test-tab-1', got '%s'", retrieved.ID)
	}

	// Close session
	if err := mgr.CloseSession("test-tab-1"); err != nil {
		t.Fatalf("Failed to close session: %v", err)
	}

	// Session should no longer exist
	if _, err := mgr.GetSession("test-tab-1"); err == nil {
		t.Errorf("Expected error getting closed session, got nil")
	}
}
