package modbus

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/logger"
	"regexp"
	"strings"
)

// extractSensorKeyCalc extracts the sensor key from a full key for calculated values
func extractSensorKeyCalc(fullKey string) string {
	parts := strings.Split(fullKey, "_")
	if len(parts) >= 2 {
		if len(parts) >= 3 {
			lastTwo := strings.Join(parts[len(parts)-2:], "_")
			knownKeys := []string{"power_active", "power_reactive", "power_apparent", "power_factor"}
			for _, known := range knownKeys {
				if lastTwo == known {
					return lastTwo
				}
			}
		}
		return parts[len(parts)-1]
	}
	return fullKey
}

// CalculatedRegisterStrategy evaluates a formula using cached register values
type CalculatedRegisterStrategy struct {
	*BaseStrategy
	evaluator    *ExpressionEvaluator
	devicePrefix string // Prefix for resolving variable names
}

// NewCalculatedRegisterStrategy creates a new calculated register strategy
func NewCalculatedRegisterStrategy(
	key string,
	register config.Register,
	devicePrefix string,
	cache *ValueCache,
) *CalculatedRegisterStrategy {
	return &CalculatedRegisterStrategy{
		BaseStrategy: &BaseStrategy{
			key:      key,
			register: register,
			cache:    cache,
		},
		evaluator:    NewExpressionEvaluator(),
		devicePrefix: devicePrefix,
	}
}

// Execute evaluates the formula and returns the calculated result
func (s *CalculatedRegisterStrategy) Execute(ctx context.Context) (*CommandResult, error) {
	if s.register.Formula == "" {
		return nil, fmt.Errorf("calculated register '%s' has no formula", s.key)
	}

	// Extract variable names from formula
	variables, err := s.extractVariables(s.register.Formula)
	if err != nil {
		return nil, fmt.Errorf("failed to extract variables from formula for '%s': %w", s.key, err)
	}

	// Fetch all variable values from cache
	variableValues := make(map[string]float64)
	logger.LogDebug("  🔍 Resolving variables for '%s':", s.key)
	for _, varName := range variables {
		// Prefix variable name with device prefix
		fullKey := fmt.Sprintf("%s_%s", s.devicePrefix, varName)

		cached, found := s.cache.Get(fullKey)
		if !found {
			logger.LogDebug("  ⚠️  Variable '%s' → '%s' NOT FOUND in cache", varName, fullKey)
			return nil, fmt.Errorf("variable '%s' (resolved to '%s') not found in cache for calculated register '%s'",
				varName, fullKey, s.key)
		}

		variableValues[varName] = cached.Value
		logger.LogDebug("  ✓ Variable '%s' → '%s' = %.2f %s", varName, fullKey, cached.Value, cached.Unit)
	}

	// Set variables in evaluator
	s.evaluator.SetVariables(variableValues)

	// Evaluate formula
	logger.LogDebug("  📐 Evaluating formula: %s", s.register.Formula)
	value, err := s.evaluator.Evaluate(s.register.Formula)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate formula for '%s': %w", s.key, err)
	}
	logger.LogDebug("  📐 Raw result: %.6f", value)

	// Apply scale factor
	value = value * s.register.ScaleFactor
	logger.LogDebug("  📐 After scale factor (%.2f): %.6f", s.register.ScaleFactor, value)

	// Create result
	result := &CommandResult{
		Strategy:    "calculated_register",
		Name:        s.register.Name,
		Value:       value,
		Unit:        s.register.Unit,
		Topic:       s.register.HATopic,
		SensorKey:   extractSensorKeyCalc(s.key), // Extract just the sensor key
		DeviceClass: s.register.DeviceClass,
		StateClass:  s.register.StateClass,
		RawData:     nil, // Calculated values have no raw data
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(s.key, result)
	}

	return result, nil
}

// extractVariables extracts variable names from a formula
// For example: "sqrt(power_active^2 + power_reactive^2)" -> ["power_active", "power_reactive"]
func (s *CalculatedRegisterStrategy) extractVariables(formula string) ([]string, error) {
	// Remove function calls to avoid matching function names
	cleaned := formula
	cleaned = regexp.MustCompile(`sqrt\(`).ReplaceAllString(cleaned, "(")
	cleaned = regexp.MustCompile(`abs\(`).ReplaceAllString(cleaned, "(")

	// Match valid variable names (alphanumeric + underscore)
	re := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\b`)
	matches := re.FindAllString(cleaned, -1)

	// Deduplicate
	seen := make(map[string]bool)
	var variables []string
	for _, match := range matches {
		// Skip if it's a number or already seen
		if !seen[match] && !isNumeric(match) {
			seen[match] = true
			variables = append(variables, match)
		}
	}

	return variables, nil
}

// isNumeric checks if a string is numeric
func isNumeric(s string) bool {
	// Simple check: if it starts with a digit, it's numeric
	if len(s) == 0 {
		return false
	}
	return s[0] >= '0' && s[0] <= '9'
}
