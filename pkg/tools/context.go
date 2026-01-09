package tools

import (
	"sync"
	"time"
)

// ToolContext provides execution context to middleware.
// It carries information about the current tool invocation and allows
// middleware to pass data between Before and After hooks.
type ToolContext struct {
	// Name is the tool being executed.
	Name ToolName

	// Input is the parsed input struct for the tool.
	Input any

	// StartTime is when execution started.
	StartTime time.Time

	// metadata stores values passed between middleware hooks.
	metadata map[string]any
	mu       sync.RWMutex
}

// NewToolContext creates a new tool context.
func NewToolContext(name ToolName, input any) *ToolContext {
	return &ToolContext{
		Name:      name,
		Input:     input,
		StartTime: time.Now(),
		metadata:  make(map[string]any),
	}
}

// Set stores a value in the context metadata.
// This is useful for passing data from Before to After hooks.
func (tc *ToolContext) Set(key string, value any) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.metadata[key] = value
}

// Get retrieves a value from the context metadata.
func (tc *ToolContext) Get(key string) (any, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	v, ok := tc.metadata[key]
	return v, ok
}

// GetString retrieves a string value from the context metadata.
func (tc *ToolContext) GetString(key string) string {
	v, ok := tc.Get(key)
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// Duration returns the time elapsed since execution started.
func (tc *ToolContext) Duration() time.Duration {
	return time.Since(tc.StartTime)
}
