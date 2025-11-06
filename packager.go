package iso8583

import (
	"encoding/json"
	"fmt"
)

// CompiledPackager holds the complete specification (schema) for an ISO8583 message.
// It contains all field configurations, bitmap encoding, length/header settings,
// and a pre-compiled validator. It is immutable and safe for concurrent use.
type CompiledPackager struct {
	fieldConfigs    map[int]FieldConfig   // Configuration for each field (DE 2, DE 3, etc.)
	bitmapEncoding  BitmapEncoding        // Binary or Hex
	lengthIndicator LengthIndicatorConfig // Config for the 2/4 byte message length prefix
	headerConfig    HeaderConfig          // Config for any message header (e.g., TPDU)
	tlvConfig       TLVConfig             // Config for TLV-encoded fields (e.g., DE 55)
	validator       *CompiledValidator    // Pre-compiled validator based on field configs
}

// NewCompiledPackager creates a new CompiledPackager from a PackagerConfig.
// It also compiles the validation rules from the config.
func NewCompiledPackager(config *PackagerConfig) *CompiledPackager {
	cp := &CompiledPackager{
		fieldConfigs:    config.Fields,
		bitmapEncoding:  config.BitmapEncoding,
		lengthIndicator: config.LengthIndicator,
		headerConfig:    config.Header,
		tlvConfig:       config.TLV,
	}

	// Pre-compile validation rules for efficiency
	cp.validator = compileValidator(config)

	return cp
}

// GetFieldConfig retrieves the configuration for a specific field number.
func (cp *CompiledPackager) GetFieldConfig(fieldNum int) (FieldConfig, bool) {
	config, exists := cp.fieldConfigs[fieldNum]
	return config, exists
}

// GetValidator returns the pre-compiled validator for this packager.
func (cp *CompiledPackager) GetValidator() *CompiledValidator {
	return cp.validator
}

// LoadPackagerFromJSON unmarshals a JSON byte slice into a PackagerConfig
// and returns a new CompiledPackager.
func LoadPackagerFromJSON(data []byte) (*CompiledPackager, error) {
	var config PackagerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse packager config: %w", err)
	}

	return NewCompiledPackager(&config), nil
}

// GetDefaultPackagerConfig returns a basic, empty packager configuration.
func DefaultPackagerConfig() *PackagerConfig {
	return &PackagerConfig{
		Fields:         DefaultConfigField, // Assumes this is a map[int]FieldConfig
		BitmapEncoding: BitmapEncodingHex,
		LengthIndicator: LengthIndicatorConfig{
			Type:   LengthIndicatorNone,
			Length: 0,
		},
		Header: HeaderConfig{
			Type:   HeaderNone,
			Length: 0,
		},
		TLV: TLVConfig{
			Type:     TLVStandard,
			Enabled:  false,
			MaxDepth: 3,
		},
	}
}

// NewPackagerConfig creates a new PackagerConfig using the options pattern.
func NewPackagerConfig(opts ...PackagerOption) *PackagerConfig {
	config := DefaultPackagerConfig()
	for _, opt := range opts {
		opt(config)
	}
	return config
}
