package iso8583

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// BitValueLength defines how to extract, validate, and format a bit field
type BitValueLength struct {
	BitNumber   int    `json:"bit_number" yaml:"bit_number"`
	DataType    string `json:"data_type" yaml:"data_type"`               // "numeric", "alpha", "alphanumeric", "alphanumeric_special"
	Length      int    `json:"length" yaml:"length"`                     // Expected length
	Padding     string `json:"padding" yaml:"padding"`                   // "left", "right", "none"
	PadChar     string `json:"pad_char" yaml:"pad_char"`                 // " " (space) or "0" (zero)
	Format      string `json:"format,omitempty" yaml:"format,omitempty"` // "YYYYMMDD", "YYYY", etc.
	From        int    `json:"from,omitempty" yaml:"from,omitempty"`
	Until       int    `json:"until,omitempty" yaml:"until,omitempty"`
	Required    bool   `json:"required" yaml:"required"`
	Alias       string `json:"alias,omitempty" yaml:"alias,omitempty"`
	TrimPadding bool   `json:"trim_padding" yaml:"trim_padding"` // Remove padding after extraction
}

// Padding constants
const (
	PaddingLeft  = "left"
	PaddingRight = "right"
	PaddingNone  = "none"
)

// Format constants
const (
	FormatYYYYMMDD = "YYYYMMDD"
	FormatYYYY     = "YYYY"
	FormatYYMMDD   = "YYMMDD"
	FormatHHMMSS   = "HHMMSS"
)

// DataType constants
const (
	DataTypeNumeric             = "numeric"              // 0-9 only
	DataTypeAlpha               = "alpha"                // A-Z, a-z only
	DataTypeAlphanumeric        = "alphanumeric"         // A-Z, a-z, 0-9
	DataTypeAlphanumericSpecial = "alphanumeric_special" // A-Z, a-z, 0-9, and special chars
	DataTypeHex                 = "hex"                  // 0-9, A-F, a-f
	DataTypeAny                 = "any"                  // No validation
)

// BitValueLengthExtractResult contains the extracted value and metadata
type BitValueLengthExtractResult struct {
	Value     string `json:"value"`
	BitNumber int    `json:"bit_number"`
	DataType  string `json:"data_type"`
	IsValid   bool   `json:"is_valid"`
	Error     string `json:"error,omitempty"`
}

func ParseLengthValue(
	isoMsg *Message,
	bitConfigs map[string]BitValueLength,
) (map[string]BitValueLengthExtractResult, error) {
	results := make(map[string]BitValueLengthExtractResult)
	var errors []string

	for key, config := range bitConfigs {
		result := BitValueLengthExtractResult{
			BitNumber: config.BitNumber,
			DataType:  config.DataType,
			IsValid:   true,
		}

		// Extract raw value from ISO message
		rawValue, err := isoMsg.GetString(config.BitNumber)
		if err != nil {
			if config.Required {
				errMsg := fmt.Sprintf("bit %d (%s): required but not found: %v",
					config.BitNumber, key, err)
				errors = append(errors, errMsg)
				result.IsValid = false
				result.Error = errMsg
				results[key] = result
				continue
			}
			continue
		}

		// Apply substring extraction if configured
		extractedValue := rawValue
		if config.From > 0 && config.Until > 0 {
			extracted, err := extractSubstring(rawValue, config.From, config.Until, key)
			if err != nil {
				errMsg := fmt.Sprintf("bit %d (%s): %v", config.BitNumber, key, err)
				errors = append(errors, errMsg)
				result.IsValid = false
				result.Error = errMsg
				results[key] = result
				continue
			}
			extractedValue = extracted
		}

		// Trim padding if configured
		if config.TrimPadding {
			extractedValue = trimPadding(extractedValue, config.Padding, config.PadChar)
		}

		// Validate format (for dates, etc.)
		if config.Format != "" {
			if err := validateFormat(extractedValue, config.Format); err != nil {
				errMsg := fmt.Sprintf("bit %d (%s): %v", config.BitNumber, key, err)
				errors = append(errors, errMsg)
				result.IsValid = false
				result.Error = errMsg
				result.Value = extractedValue
				results[key] = result
				continue
			}
		}

		// Validate data type
		if err := validateDataType(extractedValue, config.DataType); err != nil {
			errMsg := fmt.Sprintf("bit %d (%s): %v", config.BitNumber, key, err)
			errors = append(errors, errMsg)
			result.IsValid = false
			result.Error = errMsg
			result.Value = extractedValue
			results[key] = result
			continue
		}

		// Validate length (after trimming)
		if config.Length > 0 && !config.TrimPadding {
			if len(extractedValue) != config.Length {
				errMsg := fmt.Sprintf("bit %d (%s): expected length %d, got %d",
					config.BitNumber, key, config.Length, len(extractedValue))
				errors = append(errors, errMsg)
				result.IsValid = false
				result.Error = errMsg
				result.Value = extractedValue
				results[key] = result
				continue
			}
		}

		// Success
		result.Value = extractedValue
		results[key] = result
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return results, nil
}

// trimPadding removes padding characters based on configuration
func trimPadding(value, padding, padChar string) string {
	if padChar == "" {
		return value
	}

	switch padding {
	case PaddingLeft:
		// Left justified = padding on the right
		return strings.TrimRight(value, padChar)
	case PaddingRight:
		// Right justified = padding on the left
		return strings.TrimLeft(value, padChar)
	default:
		return value
	}
}

// validateFormat validates date/time formats
func validateFormat(value, format string) error {
	switch format {
	case FormatYYYYMMDD:
		if len(value) != 8 {
			return fmt.Errorf("invalid YYYYMMDD format: expected 8 digits, got %d", len(value))
		}
		_, err := time.Parse("20060102", value)
		if err != nil {
			return fmt.Errorf("invalid YYYYMMDD date: %w", err)
		}
	case FormatYYYY:
		if len(value) != 4 {
			return fmt.Errorf("invalid YYYY format: expected 4 digits, got %d", len(value))
		}
		year, err := strconv.Atoi(value)
		if err != nil || year < 1900 || year > 2100 {
			return fmt.Errorf("invalid year: %s", value)
		}
	case FormatYYMMDD:
		if len(value) != 6 {
			return fmt.Errorf("invalid YYMMDD format: expected 6 digits, got %d", len(value))
		}
	case FormatHHMMSS:
		if len(value) != 6 {
			return fmt.Errorf("invalid HHMMSS format: expected 6 digits, got %d", len(value))
		}
	}
	return nil
}

// extractSubstring extracts substring with proper bounds checking
func extractSubstring(value string, from, until int, fieldName string) (string, error) {
	if from < 1 || until < 1 {
		return "", fmt.Errorf("invalid indices: from=%d, until=%d (must be >= 1)", from, until)
	}

	if from > until {
		return "", fmt.Errorf("invalid range: from=%d > until=%d", from, until)
	}

	// Convert to 0-based indexing
	startIdx := from - 1
	endIdx := until // Fixed: should be 'until', not 'until-1' for inclusive range

	if startIdx >= len(value) {
		return "", fmt.Errorf("start index %d exceeds value length %d", from, len(value))
	}

	if endIdx > len(value) {
		return "", fmt.Errorf("end index %d exceeds value length %d", until, len(value))
	}

	return value[startIdx:endIdx], nil
}

// validateDataType validates the value against the specified data type
func validateDataType(value, dataType string) error {
	if dataType == DataTypeAny || dataType == "" {
		return nil
	}

	switch dataType {
	case DataTypeNumeric:
		return validateNumeric(value)
	case DataTypeAlpha:
		return validateAlpha(value)
	case DataTypeAlphanumeric:
		return validateAlphanumeric(value)
	case DataTypeAlphanumericSpecial:
		return validateAlphanumericSpecial(value)
	case DataTypeHex:
		return validateHex(value)
	default:
		return fmt.Errorf("unknown data type: %s", dataType)
	}
}

// validateLength checks if value length is within constraints
func validateLength(value string, minLen, maxLen int) error {
	length := len(value)

	if minLen > 0 && length < minLen {
		return fmt.Errorf("length %d is less than minimum %d", length, minLen)
	}

	if maxLen > 0 && length > maxLen {
		return fmt.Errorf("length %d exceeds maximum %d", length, maxLen)
	}

	return nil
}

// Validation functions
func validateNumeric(value string) error {
	for i, r := range value {
		if r < '0' || r > '9' {
			return fmt.Errorf("invalid numeric character '%c' at position %d", r, i)
		}
	}
	return nil
}

func validateAlpha(value string) error {
	for i, r := range value {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			return fmt.Errorf("invalid alpha character '%c' at position %d", r, i)
		}
	}
	return nil
}

func validateAlphanumeric(value string) error {
	for i, r := range value {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return fmt.Errorf("invalid alphanumeric character '%c' at position %d", r, i)
		}
	}
	return nil
}

func validateAlphanumericSpecial(value string) error {
	for i, r := range value {
		// Allow alphanumeric + common special characters
		if !((r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == ' ' || r == '-' || r == '_' || r == '.' || r == '/' ||
			r == '@' || r == '#' || r == '$' || r == '%' || r == '&' ||
			r == '*' || r == '(' || r == ')' || r == '+' || r == '=' ||
			r == ',' || r == ':' || r == ';' || r == '!' || r == '?') {
			return fmt.Errorf("invalid character '%c' at position %d", r, i)
		}
	}
	return nil
}

func validateHex(value string) error {
	for i, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'F') || (r >= 'a' && r <= 'f')) {
			return fmt.Errorf("invalid hex character '%c' at position %d", r, i)
		}
	}
	return nil
}
