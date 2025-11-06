package iso8583

import (
	"fmt"
	"regexp"
	"sync"
)

// ValidationRule defines the interface for a single validation rule.
type ValidationRule interface {
	Validate(field *Field) error
	Name() string // Returns the name of the rule (e.g., "length")
}

// CompiledValidator holds a pre-compiled set of validation rules
// derived from a PackagerConfig. It is safe for concurrent use.
type CompiledValidator struct {
	mandatoryFields map[int]bool              // Fast lookup for mandatory fields
	fieldRules      map[int][]ValidationRule  // Rules specific to a field number
	globalRules     []ValidationRule          // Rules applied to all fields
	regexCache      map[string]*regexp.Regexp // Cache for compiled regex rules
	charsetEnabled  bool                      // (Not currently used)
	lengthEnabled   bool                      // (Not currently used)
	presenceEnabled bool                      // (Not currently used)
	validationLevel ValidationLevel           // (Not currently used)
	mu              sync.RWMutex
}

// NewCompiledValidator creates a new, empty validator.
func NewCompiledValidator() *CompiledValidator {
	return &CompiledValidator{
		mandatoryFields: make(map[int]bool),
		fieldRules:      make(map[int][]ValidationRule),
		globalRules:     make([]ValidationRule, 0),
		regexCache:      make(map[string]*regexp.Regexp),
		charsetEnabled:  true,
		lengthEnabled:   true,
		presenceEnabled: true,
		validationLevel: ValidationBasic,
	}
}

// AddGlobalRule adds a rule that will be applied to all fields.
func (cv *CompiledValidator) AddGlobalRule(rule ValidationRule) {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	cv.globalRules = append(cv.globalRules, rule)
}

// ValidateMessage validates an entire Message.
// It checks for mandatory field presence and then validates all present fields.
func (cv *CompiledValidator) ValidateMessage(msg *Message, level ValidationLevel) error {
	cv.mu.RLock()
	defer cv.mu.RUnlock()

	if level == ValidationNone {
		return nil
	}

	// Check for mandatory fields
	for fieldNum := 1; fieldNum <= 128; fieldNum++ {
		if cv.mandatoryFields[fieldNum] && !msg.HasField(fieldNum) {
			return &ValidationError{
				Field:   fieldNum,
				Rule:    "mandatory",
				Message: "mandatory field missing",
			}
		}

		// Validate the field if it's present
		if msg.HasField(fieldNum) {
			field, _ := msg.GetField(fieldNum) // Error check not needed, HasField was true
			if err := cv.ValidateField(fieldNum, field); err != nil {
				return err // Return the first error found
			}
		}
	}

	return nil
}

// ValidateField validates a single field against all applicable rules.
func (cv *CompiledValidator) ValidateField(fieldNum int, field *Field) error {
	cv.mu.RLock()
	defer cv.mu.RUnlock()

	// Run field-specific rules
	if rules, exists := cv.fieldRules[fieldNum]; exists {
		for _, rule := range rules {
			if err := rule.Validate(field); err != nil {
				return &ValidationError{
					Field:   fieldNum,
					Rule:    rule.Name(),
					Message: err.Error(),
				}
			}
		}
	}

	// Run global rules
	for _, rule := range cv.globalRules {
		if err := rule.Validate(field); err != nil {
			return &ValidationError{
				Field:   fieldNum,
				Rule:    rule.Name(),
				Message: err.Error(),
			}
		}
	}

	return nil
}

// Clone creates a deep copy of the CompiledValidator.
func (cv *CompiledValidator) Clone() *CompiledValidator {
	cv.mu.RLock()
	defer cv.mu.RUnlock()

	clone := NewCompiledValidator()

	for k, v := range cv.mandatoryFields {
		clone.mandatoryFields[k] = v
	}

	for k, v := range cv.fieldRules {
		clone.fieldRules[k] = make([]ValidationRule, len(v))
		copy(clone.fieldRules[k], v)
	}

	clone.globalRules = make([]ValidationRule, len(cv.globalRules))
	copy(clone.globalRules, cv.globalRules)

	clone.charsetEnabled = cv.charsetEnabled
	clone.lengthEnabled = cv.lengthEnabled
	clone.presenceEnabled = cv.presenceEnabled
	clone.validationLevel = cv.validationLevel

	// Note: regexCache is not cloned, it will be rebuilt on demand.

	return clone
}

// --- Validation Rule Implementations ---

// LengthRule validates the field's length.
type LengthRule struct {
	MinLength   int
	MaxLength   int
	ExactLength int
	AllowEmpty  bool
}

// Name returns the rule name.
func (r *LengthRule) Name() string {
	return "length"
}

// Validate checks the field's length constraints.
func (r *LengthRule) Validate(field *Field) error {
	length := field.Length()

	if length == 0 && r.AllowEmpty {
		return nil
	}

	if r.ExactLength > 0 && length != r.ExactLength {
		return fmt.Errorf("expected length %d, got %d", r.ExactLength, length)
	}

	if r.MinLength > 0 && length < r.MinLength {
		return fmt.Errorf("length %d below minimum %d", length, r.MinLength)
	}

	if r.MaxLength > 0 && length > r.MaxLength {
		return fmt.Errorf("length %d exceeds maximum %d", length, r.MaxLength)
	}

	return nil
}

// NumericRule validates that the field contains only numeric digits.
type NumericRule struct {
	AllowEmpty        bool
	AllowLeadingZeros bool
}

// Name returns the rule name.
func (r *NumericRule) Name() string {
	return "numeric"
}

// Validate checks for non-numeric characters.
func (r *NumericRule) Validate(field *Field) error {
	data := field.Bytes()

	if len(data) == 0 && r.AllowEmpty {
		return nil
	}

	for i, b := range data {
		if b < '0' || b > '9' {
			return fmt.Errorf("non-numeric character at position %d", i)
		}
	}

	if !r.AllowLeadingZeros && len(data) > 1 && data[0] == '0' {
		return fmt.Errorf("leading zeros not allowed")
	}

	return nil
}

// AlphanumericRule validates alphanumeric content.
type AlphanumericRule struct {
	AllowEmpty        bool
	AllowSpecialChars bool   // If true, allows any printable ASCII. If false, only [0-9a-zA-Z ].
	CustomCharset     string // If set, validates against this specific charset.
}

// Name returns the rule name.
func (r *AlphanumericRule) Name() string {
	return "alphanumeric"
}

// Validate checks for invalid characters.
func (r *AlphanumericRule) Validate(field *Field) error {
	data := field.Bytes()

	if len(data) == 0 && r.AllowEmpty {
		return nil
	}

	for i, b := range data {
		if r.CustomCharset != "" {
			// Validate against custom charset
			found := false
			for _, c := range r.CustomCharset {
				if byte(c) == b {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("invalid character at position %d", i)
			}
		} else if !r.AllowSpecialChars {
			// Basic ANS validation (letters, numbers, space)
			if !((b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == ' ') {
				return fmt.Errorf("special character not allowed at position %d", i)
			}
		}
		// If AllowSpecialChars is true and CustomCharset is empty, all chars are allowed (like in field.go)
	}

	return nil
}

// BinaryRule validates binary data.
type BinaryRule struct {
	AllowEmpty        bool
	RequireEvenLength bool // Often required for hex-encoded binary data
}

// Name returns the rule name.
func (r *BinaryRule) Name() string {
	return "binary"
}

// Validate checks binary data constraints.
func (r *BinaryRule) Validate(field *Field) error {
	data := field.Bytes()

	if len(data) == 0 && r.AllowEmpty {
		return nil
	}

	if r.RequireEvenLength && len(data)%2 != 0 {
		return fmt.Errorf("binary data must have even length")
	}

	return nil
}

// RegexRule validates the field against a regular expression.
type RegexRule struct {
	Pattern     string
	AllowEmpty  bool
	Description string // User-friendly error message
	regex       *regexp.Regexp
}

// Name returns the rule name.
func (r *RegexRule) Name() string {
	return "regex"
}

// Validate checks the field against the compiled regex.
func (r *RegexRule) Validate(field *Field) error {
	if r.regex == nil {
		// Compile and cache the regex on first use
		// Note: This is not thread-safe if multiple goroutines
		// hit this simultaneously. It's better to compile this
		// in the compileValidator function.
		r.regex = regexp.MustCompile(r.Pattern)
	}

	data := field.String() // Use string for regex matching

	if len(data) == 0 && r.AllowEmpty {
		return nil
	}

	if !r.regex.MatchString(data) {
		if r.Description != "" {
			return fmt.Errorf("%s", r.Description)
		}
		return fmt.Errorf("does not match pattern %s", r.Pattern)
	}

	return nil
}

// RangeRule validates that a numeric field's value is within a given range.
type RangeRule struct {
	Min        int64
	Max        int64
	AllowEmpty bool
}

// Name returns the rule name.
func (r *RangeRule) Name() string {
	return "range"
}

// Validate parses the field as an int64 and checks the range.
func (r *RangeRule) Validate(field *Field) error {
	data := field.String()

	if len(data) == 0 && r.AllowEmpty {
		return nil
	}

	val, err := field.Int64()
	if err != nil {
		return fmt.Errorf("cannot parse as integer: %v", err)
	}

	if val < r.Min {
		return fmt.Errorf("value %d below minimum %d", val, r.Min)
	}

	if val > r.Max {
		return fmt.Errorf("value %d exceeds maximum %d", val, r.Max)
	}

	return nil
}

// CustomRule allows defining an arbitrary validation function.
type CustomRule struct {
	ValidateFunc func(*Field) error
	RuleName     string
}

// Name returns the custom rule name.
func (r *CustomRule) Name() string {
	return r.RuleName
}

// Validate executes the custom validation function.
func (r *CustomRule) Validate(field *Field) error {
	return r.ValidateFunc(field)
}

// PresenceRule validates that a field is present.
type PresenceRule struct {
	Required bool
}

// Name returns the rule name.
func (r *PresenceRule) Name() string {
	return "presence"
}

// Validate checks if the field is present.
func (r *PresenceRule) Validate(field *Field) error {
	if r.Required && !field.IsPresent() {
		return fmt.Errorf("field is required")
	}
	return nil
}

// TrackDataRule provides basic validation for track data (e.g., from a magnetic stripe).
type TrackDataRule struct {
	AllowEmpty bool
}

// Name returns the rule name.
func (r *TrackDataRule) Name() string {
	return "track_data"
}

// Validate performs a very basic check on track data.
func (r *TrackDataRule) Validate(field *Field) error {
	data := field.String()

	if len(data) == 0 && r.AllowEmpty {
		return nil
	}

	// Basic sanity check
	if len(data) < 10 {
		return fmt.Errorf("track data too short")
	}

	// More complex track data validation (e.g., format, LRC) could go here.

	return nil
}

// compileValidator creates a new CompiledValidator based on the rules
// defined in a PackagerConfig.
func compileValidator(config *PackagerConfig) *CompiledValidator {
	validator := NewCompiledValidator()

	for fieldNum, fieldConfig := range config.Fields {
		// Add mandatory presence rule
		if fieldConfig.Mandatory {
			validator.mandatoryFields[fieldNum] = true
		}

		var rules []ValidationRule

		// Add length rule
		if fieldConfig.MinLength > 0 || fieldConfig.MaxLength > 0 {
			rules = append(rules, &LengthRule{
				MinLength: fieldConfig.MinLength,
				MaxLength: fieldConfig.MaxLength,
			})
		}

		// Add content type rule
		switch fieldConfig.Type {
		case FieldTypeN:
			rules = append(rules, &NumericRule{})
		case FieldTypeANS:
			rules = append(rules, &AlphanumericRule{})
		case FieldTypeB:
			rules = append(rules, &BinaryRule{})
		}

		if len(rules) > 0 {
			validator.fieldRules[fieldNum] = rules
		}
	}

	return validator
}
