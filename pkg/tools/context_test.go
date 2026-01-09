package tools

import (
	"testing"
	"time"
)

func TestNewToolContext(t *testing.T) {
	input := QueryInput{SQL: "SELECT 1"}
	tc := NewToolContext(ToolQuery, input)

	if tc.Name != ToolQuery {
		t.Errorf("expected ToolQuery, got %v", tc.Name)
	}
	if tc.Input != input {
		t.Error("input not set correctly")
	}
	if tc.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
	if tc.metadata == nil {
		t.Error("metadata should be initialized")
	}
}

func TestToolContext_SetGet(t *testing.T) {
	tc := NewToolContext(ToolQuery, nil)

	// Test string value
	tc.Set("key1", "value1")
	val, ok := tc.Get("key1")
	if !ok {
		t.Error("expected key1 to exist")
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got %v", val)
	}

	// Test int value
	tc.Set("key2", 42)
	val, ok = tc.Get("key2")
	if !ok {
		t.Error("expected key2 to exist")
	}
	if val != 42 {
		t.Errorf("expected 42, got %v", val)
	}

	// Test non-existent key
	_, ok = tc.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent key to return false")
	}
}

func TestToolContext_GetString(t *testing.T) {
	tc := NewToolContext(ToolQuery, nil)

	// String value
	tc.Set("strKey", "hello")
	if tc.GetString("strKey") != "hello" {
		t.Error("expected 'hello'")
	}

	// Non-string value
	tc.Set("intKey", 42)
	if tc.GetString("intKey") != "" {
		t.Error("expected empty string for non-string value")
	}

	// Non-existent key
	if tc.GetString("nonexistent") != "" {
		t.Error("expected empty string for nonexistent key")
	}
}

func TestToolContext_Duration(t *testing.T) {
	tc := NewToolContext(ToolQuery, nil)

	// Wait a small amount of time
	time.Sleep(10 * time.Millisecond)

	duration := tc.Duration()
	if duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", duration)
	}
}

func TestToolContext_ConcurrentAccess(_ *testing.T) {
	tc := NewToolContext(ToolQuery, nil)

	// Run concurrent Set/Get operations
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			tc.Set("key", i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			tc.Get("key")
		}
		done <- true
	}()

	<-done
	<-done
	// If we get here without a race condition panic, the test passes
}
