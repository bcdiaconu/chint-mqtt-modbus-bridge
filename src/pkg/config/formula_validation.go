package config

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateFormula validates a formula's syntax and extracts variable names
// Returns the list of variable names used in the formula
func ValidateFormula(formula string) ([]string, error) {
	if formula == "" {
		return nil, fmt.Errorf("formula cannot be empty")
	}

	// Track variables found in the formula
	variablesMap := make(map[string]bool)

	// Remove function calls to simplify variable extraction
	// Functions: sqrt(...), abs(...)
	cleanFormula := formula

	// Extract content from functions and validate parentheses balance
	parenDepth := 0
	for _, char := range formula {
		if char == '(' {
			parenDepth++
		} else if char == ')' {
			parenDepth--
			if parenDepth < 0 {
				return nil, fmt.Errorf("unbalanced parentheses: extra closing parenthesis")
			}
		}
	}
	if parenDepth != 0 {
		return nil, fmt.Errorf("unbalanced parentheses: %d unclosed", parenDepth)
	}

	// Validate function syntax
	functionPattern := regexp.MustCompile(`(sqrt|abs)\s*\(`)
	invalidFunctions := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*\s*\(`)

	// Find all function calls
	allFunctions := invalidFunctions.FindAllString(formula, -1)
	validFunctions := functionPattern.FindAllString(formula, -1)

	// Check if there are any invalid function calls
	if len(allFunctions) != len(validFunctions) {
		// There's a function that's not sqrt or abs
		for _, fn := range allFunctions {
			fnName := strings.TrimSpace(strings.TrimSuffix(fn, "("))
			if fnName != "sqrt" && fnName != "abs" {
				return nil, fmt.Errorf("unsupported function '%s' (only sqrt and abs are supported)", fnName)
			}
		}
	}

	// Remove function calls for variable extraction
	// This is a simplified approach - we just remove "sqrt(" and "abs(" and their matching ")"
	cleanFormula = regexp.MustCompile(`(sqrt|abs)\s*\(`).ReplaceAllString(cleanFormula, "")

	// Extract variable names (alphanumeric + underscore, not starting with a digit)
	// Variable pattern: starts with letter or underscore, followed by letters, digits, or underscores
	variablePattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\b`)
	matches := variablePattern.FindAllString(cleanFormula, -1)

	for _, match := range matches {
		// Skip operators and keywords that might look like variables
		if isOperatorOrKeyword(match) {
			continue
		}
		variablesMap[match] = true
	}

	// Validate operators
	invalidOperators := regexp.MustCompile(`[^a-zA-Z0-9_\s\+\-\*/\^\(\)\.]`)
	if invalidOp := invalidOperators.FindString(formula); invalidOp != "" {
		return nil, fmt.Errorf("invalid operator or character '%s'", invalidOp)
	}

	// Check for invalid patterns like multiple operators in a row
	multipleOps := regexp.MustCompile(`[\+\-\*/\^]{2,}`)
	if multipleOps.MatchString(formula) {
		return nil, fmt.Errorf("invalid syntax: multiple operators in sequence")
	}

	// Convert map to slice
	variables := make([]string, 0, len(variablesMap))
	for varName := range variablesMap {
		variables = append(variables, varName)
	}

	if len(variables) == 0 {
		return nil, fmt.Errorf("formula contains no variables")
	}

	return variables, nil
}

// isOperatorOrKeyword checks if a string is a reserved operator or keyword
func isOperatorOrKeyword(s string) bool {
	// Reserved words that shouldn't be treated as variables
	reserved := map[string]bool{
		"sqrt": true,
		"abs":  true,
		// Future: "sin", "cos", "tan", "log", "exp", "if", "then", "else"
	}
	return reserved[s]
}
