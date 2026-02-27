package iso8583

import (
	"fmt"
	"log/slog"
	"sync"
	"unsafe"
)

// messagePool holds reusable Message objects to reduce allocations.
var messagePool = sync.Pool{
	New: func() interface{} {
		return &Message{
			// tlvData is left nil. It will be allocated on-demand
			// if TLV fields are actually parsed or set.
		}
	},
}

// Message represents a single, parsed ISO8583 message.
// It contains the MTI, header, bitmap, and all present fields.
// It is designed to be reused via a sync.Pool.
type Message struct {
	mti             [4]byte
	fields          [128]Field // Array of all possible fields
	bitmap          BitmapManager
	packager        *CompiledPackager // The specification used to parse/pack this message
	header          []byte
	tlvData         map[int][]TLV // Parsed TLV data, keyed by field number
	validationLevel ValidationLevel
	fieldPresence   [2]uint64 // Optimized bitset for field presence (1=present)
	mu              sync.RWMutex
	fullMessage     []byte // Reference to the original raw message bytes

	lastError FieldError // Stores the last error encountered during parsing
}

// NewMessage retrieves a Message from the pool and initializes it.
func NewMessage(opts ...MessageOption) *Message {
	msg := messagePool.Get().(*Message)
	msg.reset() // Ensure it's in a clean state
	for _, opt := range opts {
		opt(msg)
	}
	return msg
}

// Release returns the message to the pool for reuse.
// The message must not be used after Release is called.
func (m *Message) Release() {
	m.reset()
	messagePool.Put(m)
}

// Reset clears the message for reuse.
func (m *Message) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reset()
}

// reset is the internal, non-locking reset method.
// It clears all data, preparing the struct for reuse.
func (m *Message) reset() {
	m.mti = [4]byte{}
	m.header = nil
	m.validationLevel = ValidationNone
	m.bitmap.Reset()
	m.fieldPresence = [2]uint64{} // Clear presence bits
	m.fullMessage = nil
	m.packager = nil // Clear packager reference

	// Reset all fields
	for i := range m.fields {
		m.fields[i].reset()
	}

	// Only clear tlvData map if it was allocated.
	// This avoids allocating a map just to clear it.
	if m.tlvData != nil {
		for k := range m.tlvData {
			delete(m.tlvData, k)
		}
	}

	m.lastError.Field = 0
	m.lastError.Err = nil
}

// isFieldPresent checks the internal presence bitset for a field.
// This is faster than checking the main ISO8583 bitmap.
func (m *Message) isFieldPresent(fieldNum int) bool {
	if fieldNum < 1 || fieldNum > 128 {
		return false
	}
	idx := (fieldNum - 1) / 64 // 0 for fields 1-64, 1 for 65-128
	bit := uint64(1) << ((fieldNum - 1) % 64)
	return m.fieldPresence[idx]&bit != 0
}

// setFieldPresent sets the internal presence bit for a field.
func (m *Message) setFieldPresent(fieldNum int) {
	if fieldNum < 1 || fieldNum > 128 {
		return
	}
	idx := (fieldNum - 1) / 64
	bit := uint64(1) << ((fieldNum - 1) % 64)
	m.fieldPresence[idx] |= bit
}

// MTI returns the 4-byte Message Type Indicator.
func (m *Message) MTI() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mti[:]
}

// SetMTI sets the 4-byte Message Type Indicator.
func (m *Message) SetMTI(mti []byte) error {
	if len(mti) != 4 {
		return ErrInvalidMTI
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	copy(m.mti[:], mti)
	return nil
}

// GetField returns a pointer to the specified Field struct.
// Returns ErrFieldNotFound if the field is not present.
func (m *Message) GetField(fieldNum int) (*Field, error) {
	if fieldNum < 1 || fieldNum > 128 {
		return nil, ErrInvalidField
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isFieldPresent(fieldNum) {
		return nil, ErrFieldNotFound
	}
	return &m.fields[fieldNum-1], nil
}

// SetField sets the value of a field.
// It accepts string, []byte, int, or float64.
// Float64 values are formatted with 2 decimal places by default.
// It also sets the corresponding bit in the ISO8583 bitmap.
func (m *Message) SetField(fieldNum int, value interface{}) error {
	if fieldNum < 1 || fieldNum > 128 {
		return &FieldError{Field: fieldNum, Err: ErrInvalidField}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	field := &m.fields[fieldNum-1]

	switch v := value.(type) {
	case string:
		// Zero-copy: point field data directly to string's data
		if len(v) > 0 {
			field.data = unsafe.Slice(unsafe.StringData(v), len(v))
			field.length = len(v)
		} else {
			field.data = nil
			field.length = 0
		}
		field.fieldType = FieldTypeANS
		field.parsed = true
	case []byte:
		// Store reference to the byte slice
		field.data = v
		field.length = len(v)
		field.fieldType = FieldTypeB
		field.parsed = true
	case int:
		// Format the integer
		field.SetInt(v, FieldTypeN, 0) // SetInt handles its own locking, but we hold the message lock
	case float64:
		// Format the float with default precision of 2 decimal places
		field.SetFloat(v, FieldTypeN, 2)
	default:
		return &FieldError{Field: fieldNum, Err: fmt.Errorf("unsupported value type")}
	}

	m.setFieldPresent(fieldNum) // Update presence bitset
	m.bitmap.SetField(fieldNum) // Update ISO8583 bitmap
	return nil
}

// SetFieldWithWidth sets the value of a field with configurable width/precision.
// For int: width specifies the minimum number of digits (zero-padded if needed).
// For float: precision specifies the number of decimal places.
// It accepts string, []byte, int, or float64.
// It also sets the corresponding bit in the ISO8583 bitmap.
func (m *Message) SetFieldWithWidth(fieldNum int, value interface{}, width int) error {
	if fieldNum < 1 || fieldNum > 128 {
		return &FieldError{Field: fieldNum, Err: ErrInvalidField}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	field := &m.fields[fieldNum-1]

	switch v := value.(type) {
	case string:
		if len(v) > 0 {
			field.data = unsafe.Slice(unsafe.StringData(v), len(v))
			field.length = len(v)
		} else {
			field.data = nil
			field.length = 0
		}
		field.fieldType = FieldTypeANS
		field.parsed = true
	case []byte:
		field.data = v
		field.length = len(v)
		field.fieldType = FieldTypeB
		field.parsed = true
	case int:
		field.SetInt(v, FieldTypeN, width)
	case float64:
		field.SetFloat(v, FieldTypeN, width)
	default:
		return &FieldError{Field: fieldNum, Err: fmt.Errorf("unsupported value type")}
	}

	m.setFieldPresent(fieldNum)
	m.bitmap.SetField(fieldNum)
	return nil
}

// HasField returns true if the field is present in the message.
func (m *Message) HasField(fieldNum int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isFieldPresent(fieldNum)
}

// GetPresentFields returns a slice of all field numbers present in the message.
func (m *Message) GetPresentFields() []int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for i := 1; i <= 128; i++ {
		if m.isFieldPresent(i) {
			count++
		}
	}

	fields := make([]int, 0, count)
	for i := 1; i <= 128; i++ {
		if m.isFieldPresent(i) {
			fields = append(fields, i)
		}
	}
	return fields
}

// GetPresentFieldsInto fills the provided slice with present field numbers.
// It returns the number of fields written to the slice.
// This avoids allocating a new slice.
func (m *Message) GetPresentFieldsInto(fields []int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idx := 0
	for i := 1; i <= 128 && idx < len(fields); i++ {
		if m.isFieldPresent(i) {
			fields[idx] = i
			idx++
		}
	}
	return idx
}

// GetString is a convenience helper to get a field's value as a string.
func (m *Message) GetString(fieldNum int) (string, error) {
	field, err := m.GetField(fieldNum)
	if err != nil {
		return "", err
	}
	return field.String(), nil
}

// GetBytes is a convenience helper to get a field's value as a byte slice.
func (m *Message) GetBytes(fieldNum int) ([]byte, error) {
	field, err := m.GetField(fieldNum)
	if err != nil {
		return nil, err
	}
	return field.Bytes(), nil
}

// GetInt is a convenience helper to get a field's value as an integer.
func (m *Message) GetInt(fieldNum int) (int, error) {
	field, err := m.GetField(fieldNum)
	if err != nil {
		return 0, err
	}
	return field.Int()
}

// Unpack parses a raw byte slice into the Message struct.
// The provided data slice is referenced, not copied.
func (m *Message) Unpack(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(data) < 4 { // At least 4 bytes for MTI
		return ErrInvalidMTI
	}

	m.fullMessage = data // Store reference to original data
	offset := 0

	// 1. Parse Header (if configured)
	if m.packager != nil && m.packager.headerConfig.Type != HeaderNone {
		headerLen := m.packager.headerConfig.Length
		if len(data) < headerLen {
			return ErrInvalidHeader
		}
		m.header = data[offset : offset+headerLen]
		offset += headerLen
	}

	// 2. Parse MTI
	if len(data) < offset+4 {
		return ErrInvalidMTI
	}
	copy(m.mti[:], data[offset:offset+4])
	offset += 4

	// 3. Parse Bitmap
	encoding := BitmapEncodingHex // Default
	if m.packager != nil {
		encoding = m.packager.bitmapEncoding
	}
	bitmapLen, err := m.bitmap.UnpackBitmap(data[offset:], encoding)
	if err != nil {
		return err
	}
	offset += bitmapLen

	// 4. Parse Fields
	for fieldNum := 2; fieldNum <= 128; fieldNum++ { // Start from 2 (1 is bitmap)
		if !m.bitmap.IsFieldSet(fieldNum) {
			continue
		}

		fieldOffset, err := m.parseField(fieldNum, data, offset)
		if err != nil {
			// Store the error and return
			m.lastError.Field = fieldNum
			m.lastError.Err = err
			return &m.lastError
		}
		offset = fieldOffset
	}

	return nil
}

// parseField parses a single field from the data buffer.
// It's called by Unpack.
func (m *Message) parseField(fieldNum int, data []byte, offset int) (int, error) {
	if m.packager == nil {
		return offset, ErrNoPackagerConfigured
	}

	config, exists := m.packager.fieldConfigs[fieldNum]
	if !exists {
		return offset, ErrFieldNotConfigured
	}

	// 1. Determine the length of the field
	fieldLength, newOffset, err := calculateFieldLength(config, data, offset)
	if err != nil {
		return offset, err
	}

	// 2. Check if we have enough data
	if len(data) < newOffset+fieldLength {
		return offset, ErrInvalidLength
	}

	// 3. Slice the data and set the field
	field := &m.fields[fieldNum-1]
	field.data = data[newOffset : newOffset+fieldLength] // Zero-copy slice
	field.length = fieldLength
	field.fieldType = config.Type
	field.parsed = true

	m.setFieldPresent(fieldNum) // Update presence bitset

	return newOffset + fieldLength, nil
}

// calculateFieldLength reads the length prefix (LLVAR, LLLVAR) or uses
// the fixed length from config to determine the field's data length.
// Returns: field data length, new offset (after length prefix), error
func calculateFieldLength(config FieldConfig, data []byte, offset int) (int, int, error) {
	switch config.Length {
	case LengthFixed:
		// Fixed length, length is in MaxLength
		return config.MaxLength, offset, nil

	case LengthLLVAR:
		// 2-digit ASCII length prefix
		if len(data) < offset+2 {
			return 0, offset, ErrInvalidLength
		}
		// Fast ASCII-to-int conversion
		if data[offset] < '0' || data[offset] > '9' || data[offset+1] < '0' || data[offset+1] > '9' {
			return 0, offset, ErrInvalidLength
		}
		length := int(data[offset]-'0')*10 + int(data[offset+1]-'0')
		return length, offset + 2, nil

	case LengthLLLVAR:
		// 3-digit ASCII length prefix
		if len(data) < offset+3 {
			return 0, offset, ErrInvalidLength
		}
		if data[offset] < '0' || data[offset] > '9' ||
			data[offset+1] < '0' || data[offset+1] > '9' ||
			data[offset+2] < '0' || data[offset+2] > '9' {
			return 0, offset, ErrInvalidLength
		}
		length := int(data[offset]-'0')*100 + int(data[offset+1]-'0')*10 + int(data[offset+2]-'0')
		return length, offset + 3, nil

	case LengthLLLLVAR:
		// 4-digit ASCII length prefix
		if len(data) < offset+4 {
			return 0, offset, ErrInvalidLength
		}
		if data[offset] < '0' || data[offset] > '9' ||
			data[offset+1] < '0' || data[offset+1] > '9' ||
			data[offset+2] < '0' || data[offset+2] > '9' ||
			data[offset+3] < '0' || data[offset+3] > '9' {
			return 0, offset, ErrInvalidLength
		}
		length := int(data[offset]-'0')*1000 + int(data[offset+1]-'0')*100 +
			int(data[offset+2]-'0')*10 + int(data[offset+3]-'0')
		return length, offset + 4, nil

	default:
		return 0, offset, ErrUnsupportedLengthType
	}
}

// Pack serializes the Message struct into a byte buffer.
// Returns the total number of bytes written.
func (m *Message) Pack(buf []byte) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	offset := 0

	// 1. Pack Header (if present)
	if len(m.header) > 0 {
		if len(buf) < offset+len(m.header) {
			return 0, ErrBufferTooSmall
		}
		copy(buf[offset:], m.header)
		offset += len(m.header)
	}

	// 2. Pack MTI
	if len(buf) < offset+4 {
		return 0, ErrBufferTooSmall
	}
	copy(buf[offset:], m.mti[:])
	offset += 4

	// 3. Pack Bitmap
	encoding := BitmapEncodingHex
	if m.packager != nil {
		encoding = m.packager.bitmapEncoding
	}
	bitmapLen, err := m.bitmap.PackBitmap(buf[offset:], encoding)
	if err != nil {
		return 0, err
	}
	offset += bitmapLen

	// 4. Pack Fields
	for fieldNum := 2; fieldNum <= 128; fieldNum++ {
		if !m.isFieldPresent(fieldNum) {
			continue
		}

		fieldLen, err := m.packField(fieldNum, buf, offset)
		if err != nil {
			return 0, &FieldError{Field: fieldNum, Err: err}
		}
		offset += fieldLen
	}

	return offset, nil
}

// packField packs a single field into the buffer.
// It's called by Pack.
func (m *Message) packField(fieldNum int, buf []byte, offset int) (int, error) {
	field := &m.fields[fieldNum-1]
	if !field.parsed {
		return 0, ErrFieldNotFound
	}

	if m.packager == nil {
		return 0, fmt.Errorf("no packager configured")
	}

	config, exists := m.packager.fieldConfigs[fieldNum]
	if !exists {
		return 0, fmt.Errorf("field %d not configured", fieldNum)
	}

	fieldData := field.Bytes()
	totalLen := 0 // Total bytes written for this field (prefix + data)

	// 1. Write length prefix (LLVAR, LLLVAR, etc.)
	switch config.Length {
	case LengthLLVAR:
		if len(buf) < offset+2 {
			return 0, ErrBufferTooSmall
		}
		writeIntToASCII(buf[offset:offset+2], len(fieldData), 2)
		totalLen += 2

	case LengthLLLVAR:
		if len(buf) < offset+3 {
			return 0, ErrBufferTooSmall
		}
		writeIntToASCII(buf[offset:offset+3], len(fieldData), 3)
		totalLen += 3

	case LengthLLLLVAR:
		if len(buf) < offset+4 {
			return 0, ErrBufferTooSmall
		}
		writeIntToASCII(buf[offset:offset+4], len(fieldData), 4)
		totalLen += 4

	case LengthFixed:
		// No length prefix, but check if data length matches
		if len(fieldData) != config.MaxLength {
			return 0, fmt.Errorf("fixed field %d length mismatch: expected %d, got %d", fieldNum, config.MaxLength, len(fieldData))
		}
	}

	// 2. Write field data
	if len(buf) < offset+totalLen+len(fieldData) {
		return 0, ErrBufferTooSmall
	}
	copy(buf[offset+totalLen:], fieldData)
	totalLen += len(fieldData)

	return totalLen, nil
}

// writeIntToASCII is a fast, zero-allocation helper to format an integer
// into a byte slice with fixed-width zero padding.
func writeIntToASCII(buf []byte, val, digits int) {
	for i := digits - 1; i >= 0; i-- {
		buf[i] = byte(val%10 + '0')
		val /= 10
	}
}

// Clone creates a deep copy of the message.
// All field data is copied into new byte slices.
func (m *Message) Clone() *Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clone := NewMessage()
	clone.mti = m.mti
	clone.validationLevel = m.validationLevel
	clone.fieldPresence = m.fieldPresence
	clone.packager = m.packager // Share the immutable packager

	// Copy header
	if m.header != nil {
		clone.header = make([]byte, len(m.header))
		copy(clone.header, m.header)
	}

	// Deep copy fields
	for i := 0; i < 128; i++ {
		if m.isFieldPresent(i + 1) {
			clone.fields[i] = *m.fields[i].Clone() // Use Field.Clone for deep copy
			clone.bitmap.SetField(i + 1)
		}
	}

	// Deep copy TLV data
	if m.tlvData != nil && len(m.tlvData) > 0 {
		clone.tlvData = make(map[int][]TLV, len(m.tlvData))
		for fieldNum, tlvs := range m.tlvData {
			// Note: This is a shallow copy of the TLV slice.
			// A true deep copy would also clone the TLV structs and their Value slices.
			clone.tlvData[fieldNum] = make([]TLV, len(tlvs))
			copy(clone.tlvData[fieldNum], tlvs)
		}
	}

	return clone
}

// CreateResponse generates a response message based on the current message.
// It clones the message, flips the MTI (e.g., 0100 -> 0110),
// and sets the response code (Field 39).
func (m *Message) CreateResponse(responseCode string) (*Message, error) {
	resMsg := m.Clone()

	mti := resMsg.MTI()
	if len(mti) != 4 || mti[2] != '0' {
		return nil, fmt.Errorf("cannot create response from MTI: %s", mti)
	}

	// Flip MTI (e.g., 0x00 -> 0x10)
	mtiBytes := make([]byte, 4)
	copy(mtiBytes, mti)
	mtiBytes[2] = '1' // e.g., '0' (request) -> '1' (response)

	if err := resMsg.SetMTI(mtiBytes); err != nil {
		resMsg.Release()
		return nil, err
	}

	// Set Response Code
	err := resMsg.SetField(39, responseCode)
	if err != nil {
		return nil, err
	}

	return resMsg, nil
}

// LogValue implements the slog.LogValuer interface for structured logging.
func (m *Message) LogValue() slog.Value {
	m.mu.RLock()
	defer m.mu.RUnlock()

	attrs := make([]slog.Attr, 0, 2)
	attrs = append(attrs, slog.String("full_message", string(m.fullMessage)))
	attrs = append(attrs, slog.String("MTI", string(m.mti[:])))

	// Pre-allocate a buffer on the stack to find present fields
	var fieldsBuf [128]int
	count := 0
	for i := 1; i <= 128; i++ {
		if m.isFieldPresent(i) {
			fieldsBuf[count] = i
			count++
		}
	}

	// Build slog attributes for fields
	fieldArgs := make([]any, 0, count)
	for i := 0; i < count; i++ {
		fieldNum := fieldsBuf[i]
		field := &m.fields[fieldNum-1]
		// TODO: Add masking for sensitive fields (PAN, etc.)
		fieldArgs = append(fieldArgs, slog.String(fmt.Sprintf("%d", fieldNum), field.String()))
	}

	attrs = append(attrs, slog.Group("Fields", fieldArgs...))
	return slog.GroupValue(attrs...)
}

// Validate runs the packager's pre-compiled validator against the message.
func (m *Message) Validate() error {
	if m.packager == nil || m.packager.validator == nil {
		return nil // No validator configured
	}
	return m.packager.validator.ValidateMessage(m, m.validationLevel)
}

// SetValidationLevel sets the validation level for this message instance.
func (m *Message) SetValidationLevel(level ValidationLevel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validationLevel = level
}

// GetValidationLevel returns the current validation level.
func (m *Message) GetValidationLevel() ValidationLevel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.validationLevel
}

func (m *Message) IsNMM() bool {
	switch string(m.MTI()) {
	case MTI_NMM_REQUEST, MTI_NMM_RESPONSE:
		return true
	default:
		return false
	}
}

// MTI returns the 4-byte Message Type Indicator.
func (m *Message) GetFullMessage() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fullMessage[:]
}
