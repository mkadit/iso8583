package iso8583

import (
	"fmt"
	"strconv"
	"sync"
	"unsafe"
)

// Pre-allocated buffer pool for field operations
// Note: This pool is defined but not currently used by the Field methods in this file.
// It could be leveraged in SetInt or other formatting methods to reduce allocations.
var fieldBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 256)
		return &buf
	},
}

// reset clears the field's data, preparing it for reuse (e.g., in a message pool).
func (f *Field) reset() {
	f.data = nil
	f.length = 0
	f.fieldType = FieldTypeANS
	f.parsed = false
}

// String returns the field's data as a string.
// It performs a zero-copy conversion using unsafe.
// The resulting string is only valid as long as the underlying f.data byte slice is not modified.
func (f *Field) String() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if !f.parsed || f.data == nil {
		return ""
	}
	// Zero-copy string conversion using unsafe
	return unsafe.String(&f.data[0], f.length)
}

// Bytes returns a slice of the field's data.
// This is the raw data up to f.length.
func (f *Field) Bytes() []byte {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if !f.parsed || f.data == nil {
		return nil
	}
	return f.data[:f.length]
}

// Int parses the field's data as an integer.
// It uses a zero-copy unsafe.String conversion to avoid allocations.
func (f *Field) Int() (int, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if !f.parsed || f.data == nil {
		return 0, ErrFieldNotFound
	}
	// Zero-copy using unsafe.String
	return strconv.Atoi(unsafe.String(&f.data[0], f.length))
}

// Int64 parses the field's data as an int64.
// It uses a zero-copy unsafe.String conversion to avoid allocations.
func (f *Field) Int64() (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if !f.parsed || f.data == nil {
		return 0, ErrFieldNotFound
	}
	return strconv.ParseInt(unsafe.String(&f.data[0], f.length), 10, 64)
}

// Length returns the length of the field's data in bytes.
func (f *Field) Length() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.length
}

// Type returns the configured FieldType (N, ANS, B, etc.).
func (f *Field) Type() FieldType {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.fieldType
}

// IsPresent returns true if the field has been successfully parsed or set.
func (f *Field) IsPresent() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.parsed && f.data != nil
}

// SetString sets the field's value from a string.
// This is a zero-copy operation. The field's internal data slice
// will point directly to the string's underlying data.
// This is unsafe and should only be used if the string's lifetime
// is guaranteed to exceed the field's.
func (f *Field) SetString(value string, fieldType FieldType) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Zero-copy: store pointer to string data directly
	f.data = unsafe.Slice(unsafe.StringData(value), len(value))
	f.length = len(value)
	f.fieldType = fieldType
	f.parsed = true
}

// SetBytes sets the field's value from a byte slice.
// The field will hold a reference to the provided slice, not a copy.
func (f *Field) SetBytes(value []byte, fieldType FieldType) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = value
	f.length = len(value)
	f.fieldType = fieldType
	f.parsed = true
}

// SetInt sets the field's value from an integer.
// It formats the integer into a byte slice, applying zero-padding if width > 0.
// It attempts to reuse the existing f.data buffer if capacity allows,
// otherwise, a new slice is allocated.
func (f *Field) SetInt(value int, fieldType FieldType, width int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Use stack buffer for small integers to avoid heap allocation
	var stackBuf [20]byte // 20 digits is enough for a 64-bit int
	n := formatIntToBytes(stackBuf[:], value, width)

	// Reuse existing buffer if capacity is sufficient
	if cap(f.data) >= n {
		f.data = f.data[:n]
		copy(f.data, stackBuf[:n])
	} else {
		// Allocate a new buffer if capacity is too small
		f.data = make([]byte, n)
		copy(f.data, stackBuf[:n])
	}

	f.length = n
	f.fieldType = fieldType
	f.parsed = true
}

// formatIntToBytes converts an integer to its ASCII representation in the buffer.
// It applies zero-padding to the left if the specified width is larger than
// the number of digits.
func formatIntToBytes(buf []byte, value int, width int) int {
	if value == 0 {
		if width > 0 {
			for i := 0; i < width; i++ {
				buf[i] = '0'
			}
			return width
		}
		buf[0] = '0'
		return 1
	}

	// Write digits backwards from the end of the buffer
	i := len(buf) - 1
	for value > 0 {
		buf[i] = byte(value%10 + '0')
		value /= 10
		i--
	}

	digits := len(buf) - 1 - i
	if width > digits {
		// Need padding
		padding := width - digits
		// Shift digits to the right to make space for padding
		copy(buf[padding:], buf[i+1:])
		// Add padding
		for j := 0; j < padding; j++ {
			buf[j] = '0'
		}
		return width
	}

	// No padding needed, just move digits to the start of the buffer
	copy(buf, buf[i+1:])
	return digits
}

// Validate checks the field's data against the constraints in FieldConfig.
func (f *Field) Validate(config FieldConfig) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Check for presence
	if !f.parsed {
		if config.Mandatory {
			return fmt.Errorf("mandatory field not present")
		}
		return nil // Optional and not present is valid
	}

	// Check length
	if config.MinLength > 0 && f.length < config.MinLength {
		return fmt.Errorf("field length %d below minimum %d", f.length, config.MinLength)
	}

	if config.MaxLength > 0 && f.length > config.MaxLength {
		return fmt.Errorf("field length %d exceeds maximum %d", f.length, config.MaxLength)
	}

	// Check content type
	switch config.Type {
	case FieldTypeN:
		return f.validateNumeric()
	case FieldTypeANS:
		return f.validateAlphanumeric()
	case FieldTypeB:
		return f.validateBinary()
	}

	return nil
}

// validateNumeric checks if the field contains only numeric digits ('0'-'9').
func (f *Field) validateNumeric() error {
	for i := 0; i < f.length; i++ {
		if f.data[i] < '0' || f.data[i] > '9' {
			return fmt.Errorf("non-numeric character at position %d", i)
		}
	}
	return nil
}

// validateAlphanumeric checks if the field contains only printable ASCII characters (32-126).
func (f *Field) validateAlphanumeric() error {
	for i := 0; i < f.length; i++ {
		if f.data[i] < 32 || f.data[i] > 126 { // Basic printable ASCII
			return fmt.Errorf("invalid character at position %d", i)
		}
	}
	return nil
}

// validateBinary performs validation for binary fields (currently a no-op).
func (f *Field) validateBinary() error {
	// Binary data can contain any byte, so no validation is needed by default.
	return nil
}

// Clone creates a deep copy of the field.
// It allocates a new byte slice and copies the data.
func (f *Field) Clone() *Field {
	f.mu.RLock()
	defer f.mu.RUnlock()

	clone := &Field{
		length:    f.length,
		fieldType: f.fieldType,
		parsed:    f.parsed,
	}

	if f.data != nil {
		// Create a new slice and copy the data
		clone.data = make([]byte, f.length)
		copy(clone.data, f.data[:f.length])
	}

	return clone
}
