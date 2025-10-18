package unit

import (
	"mqtt-modbus-bridge/pkg/config"
	"testing"
)

func TestValidateFormula(t *testing.T) {
	tests := []struct {
		name          string
		formula       string
		wantVariables []string
		wantError     bool
		errorContains string
	}{
		{
			name:          "Simple addition",
			formula:       "power_active + power_reactive",
			wantVariables: []string{"power_active", "power_reactive"},
			wantError:     false,
		},
		{
			name:          "Power triangle calculation",
			formula:       "sqrt(power_apparent^2 - power_active^2)",
			wantVariables: []string{"power_apparent", "power_active"},
			wantError:     false,
		},
		{
			name:          "Multiple operations",
			formula:       "sqrt(power_active^2 + power_reactive^2)",
			wantVariables: []string{"power_active", "power_reactive"},
			wantError:     false,
		},
		{
			name:          "Absolute value",
			formula:       "abs(power_factor)",
			wantVariables: []string{"power_factor"},
			wantError:     false,
		},
		{
			name:          "Complex formula",
			formula:       "sqrt(abs(voltage_L1^2 + voltage_L2^2) + voltage_L3^2)",
			wantVariables: []string{"voltage_L1", "voltage_L2", "voltage_L3"},
			wantError:     false,
		},
		{
			name:          "Division",
			formula:       "energy_total / 1000",
			wantVariables: []string{"energy_total"},
			wantError:     false,
		},
		{
			name:          "Multiplication",
			formula:       "voltage * current",
			wantVariables: []string{"voltage", "current"},
			wantError:     false,
		},
		{
			name:          "Underscores in variable names",
			formula:       "power_active_L1 + power_active_L2 + power_active_L3",
			wantVariables: []string{"power_active_L1", "power_active_L2", "power_active_L3"},
			wantError:     false,
		},
		// Error cases
		{
			name:          "Empty formula",
			formula:       "",
			wantError:     true,
			errorContains: "cannot be empty",
		},
		{
			name:          "Unbalanced parentheses - extra opening",
			formula:       "sqrt(power_active + power_reactive",
			wantError:     true,
			errorContains: "unbalanced parentheses",
		},
		{
			name:          "Unbalanced parentheses - extra closing",
			formula:       "power_active + power_reactive)",
			wantError:     true,
			errorContains: "unbalanced parentheses",
		},
		{
			name:          "Unsupported function",
			formula:       "sin(power_active)",
			wantError:     true,
			errorContains: "unsupported function 'sin'",
		},
		{
			name:          "Invalid operator",
			formula:       "power_active & power_reactive",
			wantError:     true,
			errorContains: "invalid operator",
		},
		{
			name:          "Multiple operators in sequence",
			formula:       "power_active ++ power_reactive",
			wantError:     true,
			errorContains: "multiple operators",
		},
		{
			name:          "No variables (only numbers)",
			formula:       "123 + 456",
			wantError:     true,
			errorContains: "no variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variables, err := config.ValidateFormula(tt.formula)

			// Check error expectation
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateFormula() expected error but got none")
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("ValidateFormula() error = %v, want error containing %q", err, tt.errorContains)
				}
				return
			}

			// No error expected
			if err != nil {
				t.Errorf("ValidateFormula() unexpected error = %v", err)
				return
			}

			// Check variables
			if len(variables) != len(tt.wantVariables) {
				t.Errorf("ValidateFormula() got %d variables, want %d\nGot: %v\nWant: %v",
					len(variables), len(tt.wantVariables), variables, tt.wantVariables)
				return
			}

			// Check each variable is present (order doesn't matter)
			varMap := make(map[string]bool)
			for _, v := range variables {
				varMap[v] = true
			}
			for _, want := range tt.wantVariables {
				if !varMap[want] {
					t.Errorf("ValidateFormula() missing variable %q\nGot: %v\nWant: %v",
						want, variables, tt.wantVariables)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(s) > len(substr)+1 && containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
