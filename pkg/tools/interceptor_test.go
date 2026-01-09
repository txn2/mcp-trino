package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestQueryInterceptorFunc(t *testing.T) {
	interceptor := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql + " LIMIT 10", nil
	})

	result, err := interceptor.Intercept(context.Background(), "SELECT * FROM t", ToolQuery)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "SELECT * FROM t LIMIT 10" {
		t.Errorf("expected transformed SQL, got: %s", result)
	}
}

func TestQueryInterceptorFunc_Error(t *testing.T) {
	expectedErr := errors.New("rejected")
	interceptor := QueryInterceptorFunc(func(_ context.Context, _ string, _ ToolName) (string, error) {
		return "", expectedErr
	})

	_, err := interceptor.Intercept(context.Background(), "SELECT * FROM t", ToolQuery)
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestInterceptorChain_Intercept(t *testing.T) {
	// First interceptor adds WHERE.
	i1 := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql + " WHERE 1=1", nil
	})

	// Second interceptor adds LIMIT.
	i2 := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql + " LIMIT 100", nil
	})

	chain := NewInterceptorChain(i1, i2)

	result, err := chain.Intercept(context.Background(), "SELECT * FROM t", ToolQuery)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "SELECT * FROM t WHERE 1=1 LIMIT 100" {
		t.Errorf("expected chained transformations, got: %s", result)
	}
}

func TestInterceptorChain_Intercept_Error(t *testing.T) {
	expectedErr := errors.New("rejected")

	i1 := QueryInterceptorFunc(func(_ context.Context, _ string, _ ToolName) (string, error) {
		return "", expectedErr
	})

	i2 := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		t.Error("i2 should not be called")
		return sql, nil
	})

	chain := NewInterceptorChain(i1, i2)

	_, err := chain.Intercept(context.Background(), "SELECT * FROM t", ToolQuery)
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestInterceptorChain_Append(t *testing.T) {
	chain := NewInterceptorChain()
	if chain.Len() != 0 {
		t.Errorf("expected empty chain, got %d", chain.Len())
	}

	interceptor := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql, nil
	})
	chain.Append(interceptor)

	if chain.Len() != 1 {
		t.Errorf("expected chain length 1, got %d", chain.Len())
	}
}

func TestInterceptorChain_ToolNamePassed(t *testing.T) {
	var receivedToolName ToolName

	interceptor := QueryInterceptorFunc(func(_ context.Context, sql string, toolName ToolName) (string, error) {
		receivedToolName = toolName
		return sql, nil
	})

	chain := NewInterceptorChain(interceptor)
	_, _ = chain.Intercept(context.Background(), "SELECT 1", ToolExplain)

	if receivedToolName != ToolExplain {
		t.Errorf("expected ToolExplain, got %v", receivedToolName)
	}
}

// TestInterceptor_SQLValidation tests SQL validation interceptor example.
func TestInterceptor_SQLValidation(t *testing.T) {
	validator := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		upperSQL := strings.ToUpper(sql)
		if strings.Contains(upperSQL, "DROP") {
			return "", errors.New("DROP statements are not allowed")
		}
		return sql, nil
	})

	// Valid query.
	result, err := validator.Intercept(context.Background(), "SELECT * FROM users", ToolQuery)
	if err != nil {
		t.Errorf("unexpected error for valid query: %v", err)
	}
	if result != "SELECT * FROM users" {
		t.Errorf("query should be unchanged: %s", result)
	}

	// Invalid query.
	_, err = validator.Intercept(context.Background(), "DROP TABLE users", ToolQuery)
	if err == nil {
		t.Error("expected error for DROP statement")
	}
}
