package modbus

import (
	"fmt"
	"math"
	"mqtt-modbus-bridge/pkg/logger"
	"regexp"
	"strconv"
	"strings"
)

// ExpressionEvaluator evaluates mathematical expressions with variable substitution
type ExpressionEvaluator struct {
	variables map[string]float64
}

// NewExpressionEvaluator creates a new expression evaluator
func NewExpressionEvaluator() *ExpressionEvaluator {
	return &ExpressionEvaluator{
		variables: make(map[string]float64),
	}
}

// SetVariable sets a variable value for evaluation
func (e *ExpressionEvaluator) SetVariable(name string, value float64) {
	e.variables[name] = value
}

// SetVariables sets multiple variables at once
func (e *ExpressionEvaluator) SetVariables(vars map[string]float64) {
	for name, value := range vars {
		e.variables[name] = value
	}
}

// Evaluate evaluates a mathematical expression
// Supported operations: +, -, *, /, ^(power), sqrt(), abs()
// Examples:
//   - "power_active + power_reactive"
//   - "sqrt(power_active^2 + power_reactive^2)"
//   - "abs(power_factor)"
func (e *ExpressionEvaluator) Evaluate(expression string) (float64, error) {
	if expression == "" {
		return 0, fmt.Errorf("empty expression")
	}

	// Replace variables with their values
	expr := expression
	for varName, varValue := range e.variables {
		// Replace variable names with their numeric values
		// Use word boundaries to avoid partial matches (e.g., "power" shouldn't match "power_active")
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(varName))
		re := regexp.MustCompile(pattern)
		oldExpr := expr
		expr = re.ReplaceAllString(expr, fmt.Sprintf("%f", varValue))
		if oldExpr != expr {
			logger.LogDebug("    🔄 Replaced '%s' with %.6f → %s", varName, varValue, expr)
		}
	}

	logger.LogDebug("    📐 Final expression to evaluate: %s", expr)

	// Evaluate the expression
	result, err := e.evaluateNumericExpression(expr)
	if err != nil {
		return 0, fmt.Errorf("error evaluating expression '%s': %w", expression, err)
	}

	return result, nil
}

// evaluateNumericExpression evaluates a numeric expression
func (e *ExpressionEvaluator) evaluateNumericExpression(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)

	// Handle function calls first
	if strings.Contains(expr, "sqrt(") {
		return e.evaluateSqrt(expr)
	}
	if strings.Contains(expr, "abs(") {
		return e.evaluateAbs(expr)
	}

	// Handle parentheses first (highest precedence after functions)
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return e.evaluateNumericExpression(expr[1 : len(expr)-1])
	}

	// Handle addition and subtraction (lowest precedence)
	if idx := e.findOperator(expr, "+-"); idx != -1 {
		left, err := e.evaluateNumericExpression(expr[:idx])
		if err != nil {
			return 0, err
		}
		right, err := e.evaluateNumericExpression(expr[idx+1:])
		if err != nil {
			return 0, err
		}

		if expr[idx] == '+' {
			return left + right, nil
		}
		return left - right, nil
	}

	// Handle multiplication and division
	if idx := e.findOperator(expr, "*/"); idx != -1 {
		left, err := e.evaluateNumericExpression(expr[:idx])
		if err != nil {
			return 0, err
		}
		right, err := e.evaluateNumericExpression(expr[idx+1:])
		if err != nil {
			return 0, err
		}

		if expr[idx] == '*' {
			return left * right, nil
		}
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	}

	// Handle power operator (^) - highest precedence among binary operators
	if idx := e.findOperator(expr, "^"); idx != -1 {
		left, err := e.evaluateNumericExpression(expr[:idx])
		if err != nil {
			return 0, err
		}
		right, err := e.evaluateNumericExpression(expr[idx+1:])
		if err != nil {
			return 0, err
		}

		return math.Pow(left, right), nil
	}

	// Parse as number
	value, err := strconv.ParseFloat(expr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: '%s'", expr)
	}

	return value, nil
}

// evaluateSqrt evaluates sqrt() function
func (e *ExpressionEvaluator) evaluateSqrt(expr string) (float64, error) {
	// Find sqrt(...) pattern
	re := regexp.MustCompile(`sqrt\(([^)]+)\)`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid sqrt expression")
	}

	inner := matches[1]
	logger.LogDebug("    🔧 sqrt inner expression: '%s'", inner)

	innerValue, err := e.evaluateNumericExpression(inner)
	if err != nil {
		logger.LogDebug("    ❌ Error evaluating sqrt inner: %v", err)
		return 0, err
	}

	logger.LogDebug("    🔧 sqrt inner value: %.6f", innerValue)

	if innerValue < 0 {
		return 0, fmt.Errorf("sqrt of negative number: %.6f", innerValue)
	}

	result := math.Sqrt(innerValue)
	logger.LogDebug("    🔧 sqrt result: %.6f", result)

	// Replace sqrt(...) with the result and continue evaluation
	remaining := strings.Replace(expr, matches[0], fmt.Sprintf("%f", result), 1)
	if remaining == fmt.Sprintf("%f", result) {
		return result, nil
	}

	return e.evaluateNumericExpression(remaining)
}

// evaluateAbs evaluates abs() function
func (e *ExpressionEvaluator) evaluateAbs(expr string) (float64, error) {
	// Find abs(...) pattern
	re := regexp.MustCompile(`abs\(([^)]+)\)`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid abs expression")
	}

	inner := matches[1]
	innerValue, err := e.evaluateNumericExpression(inner)
	if err != nil {
		return 0, err
	}

	result := math.Abs(innerValue)

	// Replace abs(...) with the result and continue evaluation
	remaining := strings.Replace(expr, matches[0], fmt.Sprintf("%f", result), 1)
	if remaining == fmt.Sprintf("%f", result) {
		return result, nil
	}

	return e.evaluateNumericExpression(remaining)
}

// findOperator finds the position of an operator outside of parentheses
func (e *ExpressionEvaluator) findOperator(expr string, operators string) int {
	depth := 0
	for i := len(expr) - 1; i >= 0; i-- {
		switch expr[i] {
		case ')':
			depth++
		case '(':
			depth--
		default:
			if depth == 0 && strings.ContainsRune(operators, rune(expr[i])) {
				return i
			}
		}
	}
	return -1
}
