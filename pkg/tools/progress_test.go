package tools

import (
	"context"
	"sync/atomic"
	"testing"
)

// testNotifier records calls for testing.
type testNotifier struct {
	calls atomic.Int32
}

func (n *testNotifier) Notify(_ context.Context, _, _ float64, _ string) error {
	n.calls.Add(1)
	return nil
}

func TestWithProgressNotifier_RoundTrip(t *testing.T) {
	notifier := &testNotifier{}
	ctx := WithProgressNotifier(context.Background(), notifier)

	got := GetProgressNotifier(ctx)
	if got == nil {
		t.Fatal("expected notifier, got nil")
	}

	if err := got.Notify(ctx, 1, 3, "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notifier.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", notifier.calls.Load())
	}
}

func TestGetProgressNotifier_NilContext(t *testing.T) {
	// No notifier in context should return nil.
	got := GetProgressNotifier(context.Background())
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestNotifyProgress_NilNotifier(_ *testing.T) {
	// Should not panic with nil notifier.
	notifyProgress(context.Background(), nil, 0, 3, "test")
}

func TestNotifyProgress_WithNotifier(t *testing.T) {
	n := &testNotifier{}
	notifyProgress(context.Background(), n, 1, 3, "step 1")
	if n.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", n.calls.Load())
	}
}

func TestWithProgressNotifier_Overwrite(t *testing.T) {
	n1 := &testNotifier{}
	n2 := &testNotifier{}

	ctx := WithProgressNotifier(context.Background(), n1)
	ctx = WithProgressNotifier(ctx, n2)

	got := GetProgressNotifier(ctx)
	if got == nil {
		t.Fatal("expected notifier, got nil")
	}

	// Should be n2, not n1.
	_ = got.Notify(ctx, 1, 1, "check") //nolint:errcheck // test
	if n1.calls.Load() != 0 {
		t.Fatal("first notifier should not have been called")
	}
	if n2.calls.Load() != 1 {
		t.Fatal("second notifier should have been called")
	}
}
