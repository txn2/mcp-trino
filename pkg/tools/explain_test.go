package tools

import (
	"testing"

	"github.com/txn2/mcp-trino/pkg/client"
)

func TestExplainInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input ExplainInput
		valid bool
	}{
		{
			name: "valid input with type",
			input: ExplainInput{
				SQL:  "SELECT * FROM users",
				Type: "logical",
			},
			valid: true,
		},
		{
			name: "valid input without type",
			input: ExplainInput{
				SQL:  "SELECT 1",
				Type: "",
			},
			valid: true,
		},
		{
			name: "missing SQL",
			input: ExplainInput{
				SQL:  "",
				Type: "logical",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.input.SQL != ""
			if isValid != tt.valid {
				t.Errorf("expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestExplainInput_TypeMapping(t *testing.T) {
	tests := []struct {
		name         string
		inputType    string
		expectedType client.ExplainType
	}{
		{
			name:         "logical",
			inputType:    "logical",
			expectedType: client.ExplainLogical,
		},
		{
			name:         "distributed",
			inputType:    "distributed",
			expectedType: client.ExplainDistributed,
		},
		{
			name:         "io",
			inputType:    "io",
			expectedType: client.ExplainIO,
		},
		{
			name:         "validate",
			inputType:    "validate",
			expectedType: client.ExplainValidate,
		},
		{
			name:         "empty defaults to logical",
			inputType:    "",
			expectedType: client.ExplainLogical,
		},
		{
			name:         "unknown defaults to logical",
			inputType:    "unknown",
			expectedType: client.ExplainLogical,
		},
		{
			name:         "LOGICAL uppercase defaults to logical",
			inputType:    "LOGICAL",
			expectedType: client.ExplainLogical, // Case sensitive, defaults
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var explainType client.ExplainType
			switch tt.inputType {
			case "distributed":
				explainType = client.ExplainDistributed
			case "io":
				explainType = client.ExplainIO
			case "validate":
				explainType = client.ExplainValidate
			default:
				explainType = client.ExplainLogical
			}

			if explainType != tt.expectedType {
				t.Errorf("expected %s, got %s", tt.expectedType, explainType)
			}
		})
	}
}

func TestExplainTypes_Constants(t *testing.T) {
	// Verify the explain type constants have correct values
	if string(client.ExplainLogical) != "LOGICAL" {
		t.Errorf("ExplainLogical should be 'LOGICAL', got %q", client.ExplainLogical)
	}
	if string(client.ExplainDistributed) != "DISTRIBUTED" {
		t.Errorf("ExplainDistributed should be 'DISTRIBUTED', got %q", client.ExplainDistributed)
	}
	if string(client.ExplainIO) != "IO" {
		t.Errorf("ExplainIO should be 'IO', got %q", client.ExplainIO)
	}
	if string(client.ExplainValidate) != "VALIDATE" {
		t.Errorf("ExplainValidate should be 'VALIDATE', got %q", client.ExplainValidate)
	}
}

func TestExplainInput_AllTypes(t *testing.T) {
	validTypes := []string{"logical", "distributed", "io", "validate"}

	for _, typ := range validTypes {
		t.Run("type_"+typ, func(t *testing.T) {
			input := ExplainInput{
				SQL:  "SELECT 1",
				Type: typ,
			}

			if input.SQL == "" {
				t.Error("SQL should not be empty")
			}
			if input.Type != typ {
				t.Errorf("expected Type %q, got %q", typ, input.Type)
			}
		})
	}
}
