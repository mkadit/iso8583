package iso8583

// MessageOption represents a functional option for message configuration
type MessageOption func(*Message)

// WithPackager sets the packager for the message
func WithPackager(packager *CompiledPackager) MessageOption {
	return func(m *Message) {
		m.packager = packager
	}
}

// WithHeader sets the header for a message
func WithHeader(header []byte) MessageOption {
	return func(m *Message) {
		m.header = make([]byte, len(header))
		copy(m.header, header)
	}
}

// WithMTI sets the Message Type Indicator
func WithMTI(mti []byte) MessageOption {
	return func(m *Message) {
		m.SetMTI(mti)
	}
}

// WithTLVSupport enables TLV processing
func WithTLVSupport(enabled bool) MessageOption {
	return func(m *Message) {
		if enabled && m.tlvData == nil {
			m.tlvData = make(map[int][]TLV)
		}
	}
}

// WithField sets a field value during message creation
func WithField(fieldNum int, value interface{}) MessageOption {
	return func(m *Message) {
		m.SetField(fieldNum, value)
	}
}

// WithFields sets multiple fields during message creation
func WithFields(fields map[int]interface{}) MessageOption {
	return func(m *Message) {
		for fieldNum, value := range fields {
			m.SetField(fieldNum, value)
		}
	}
}

// PackagerOption represents a functional option for packager configuration
type PackagerOption func(*PackagerConfig)

// WithFieldConfig adds a field configuration
func WithFieldConfig(fieldNum int, config FieldConfig) PackagerOption {
	return func(pc *PackagerConfig) {
		if pc.Fields == nil {
			pc.Fields = make(map[int]FieldConfig)
		}
		pc.Fields[fieldNum] = config
	}
}

// WithHeaderConfig sets the header configuration
func WithHeaderConfig(config HeaderConfig) PackagerOption {
	return func(pc *PackagerConfig) {
		pc.Header = config
	}
}

// WithTLVConfig sets the TLV configuration
func WithTLVConfig(config TLVConfig) PackagerOption {
	return func(pc *PackagerConfig) {
		pc.TLV = config
	}
}

// Validation-related options
func WithValidationLevel(level ValidationLevel) MessageOption {
	return func(m *Message) {
		m.validationLevel = level
	}
}

func WithStrictValidation() MessageOption {
	return WithValidationLevel(ValidationStrict)
}

func WithBasicValidation() MessageOption {
	return WithValidationLevel(ValidationBasic)
}

func WithCustomValidation(rules ...ValidationRule) MessageOption {
	return func(m *Message) {
		m.validationLevel = ValidationCustom
		if m.packager != nil && m.packager.validator != nil {
			for _, rule := range rules {
				m.packager.validator.AddGlobalRule(rule)
			}
		}
	}
}
