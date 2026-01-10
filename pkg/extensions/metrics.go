package extensions

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/tools"
)

// MetricsCollector defines the interface for collecting metrics.
// Implement this interface to integrate with your metrics system (Prometheus, StatsD, etc.).
type MetricsCollector interface {
	// IncCounter increments a counter metric.
	IncCounter(name string, labels map[string]string)

	// ObserveDuration records a duration observation.
	ObserveDuration(name string, d time.Duration, labels map[string]string)
}

// MetricsMiddleware collects metrics for tool calls.
// It demonstrates the Before/After pattern for timing operations.
type MetricsMiddleware struct {
	collector MetricsCollector
}

// NewMetricsMiddleware creates a new metrics middleware with the given collector.
func NewMetricsMiddleware(collector MetricsCollector) *MetricsMiddleware {
	return &MetricsMiddleware{collector: collector}
}

// Before is a no-op; timing is handled via ToolContext.StartTime.
func (mm *MetricsMiddleware) Before(ctx context.Context, _ *tools.ToolContext) (context.Context, error) {
	return ctx, nil
}

// After records metrics for the completed tool call.
func (mm *MetricsMiddleware) After(
	_ context.Context,
	tc *tools.ToolContext,
	result *mcp.CallToolResult,
	handlerErr error,
) (*mcp.CallToolResult, error) {
	labels := map[string]string{
		"tool": string(tc.Name),
	}

	// Determine status
	status := "success"
	if handlerErr != nil {
		status = "error"
	} else if result != nil && result.IsError {
		status = "failed"
	}
	labels["status"] = status

	// Record call count
	mm.collector.IncCounter("mcp_trino_tool_calls_total", labels)

	// Record duration
	mm.collector.ObserveDuration("mcp_trino_tool_duration_seconds", tc.Duration(), labels)

	return result, handlerErr
}

// Verify MetricsMiddleware implements ToolMiddleware.
var _ tools.ToolMiddleware = (*MetricsMiddleware)(nil)

// InMemoryCollector is a simple in-memory metrics collector for testing and debugging.
// For production use, implement MetricsCollector with your preferred metrics system.
type InMemoryCollector struct {
	mu        sync.RWMutex
	counters  map[string]int64
	durations map[string][]time.Duration
}

// NewInMemoryCollector creates a new in-memory metrics collector.
func NewInMemoryCollector() *InMemoryCollector {
	return &InMemoryCollector{
		counters:  make(map[string]int64),
		durations: make(map[string][]time.Duration),
	}
}

// IncCounter increments a counter.
func (c *InMemoryCollector) IncCounter(name string, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := formatMetricKey(name, labels)
	c.counters[key]++
}

// ObserveDuration records a duration observation.
func (c *InMemoryCollector) ObserveDuration(name string, d time.Duration, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := formatMetricKey(name, labels)
	c.durations[key] = append(c.durations[key], d)
}

// GetCounter returns the current value of a counter.
func (c *InMemoryCollector) GetCounter(name string, labels map[string]string) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := formatMetricKey(name, labels)
	return c.counters[key]
}

// GetDurations returns all recorded durations for a metric.
func (c *InMemoryCollector) GetDurations(name string, labels map[string]string) []time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := formatMetricKey(name, labels)
	// Return a copy to avoid race conditions
	result := make([]time.Duration, len(c.durations[key]))
	copy(result, c.durations[key])
	return result
}

// Reset clears all collected metrics.
func (c *InMemoryCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters = make(map[string]int64)
	c.durations = make(map[string][]time.Duration)
}

// formatMetricKey creates a unique key from metric name and labels.
func formatMetricKey(name string, labels map[string]string) string {
	key := name

	// Sort labels for consistent key generation
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		key += "|" + k + "=" + labels[k]
	}
	return key
}

// Verify InMemoryCollector implements MetricsCollector.
var _ MetricsCollector = (*InMemoryCollector)(nil)
