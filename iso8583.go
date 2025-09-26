// Package iso8583 provides zero-allocation, high-performance ISO8583 message processing.
// This package is designed for financial transaction processing where memory efficiency
// and zero garbage collection pressure are critical.
//
// The package avoids internal object pooling, allowing applications to manage
// memory allocation strategies according to their specific needs.
//
// Example usage:
//
//	packager := iso8583.NewStandardPackager()
//	var msg iso8583.Message
//
//	// Parse incoming message
//	if err := msg.Parse(packager, data); err != nil {
//		return err
//	}
//
//	// Process and pack response
//	var response iso8583.Message
//	response.SetMTIString("0210")
//	response.CopyRequiredFields(&msg)
//
//	buf := make([]byte, 4096)
//	length, err := response.Pack(packager, buf)
package iso8583

import (
	"errors"
	"unsafe"
)

// Package constants
const (
	Version   = "2.0.0"
	MaxFields = 128
	MaxMTILen = 4
	BitmapLen = 16
	MaxMsgLen = 65536
	HeaderLen = 2
)

// Common errors
var (
	ErrInvalidLength    = errors.New("iso8583: invalid length")
	ErrInsufficientData = errors.New("iso8583: insufficient data")
	ErrInvalidMTI       = errors.New("iso8583: invalid MTI")
	ErrInvalidBit       = errors.New("iso8583: invalid bit number")
	ErrInvalidPackager  = errors.New("iso8583: invalid packager value")
	ErrBufferTooSmall   = errors.New("iso8583: buffer too small")
	ErrFieldNotFound    = errors.New("iso8583: field not found")
	ErrInvalidBitmap    = errors.New("iso8583: invalid bitmap")
)

// Field length specifications
const (
	LLVAR  = -2 // Length-Length-Variable (2 digit length)
	LLLVAR = -3 // Length-Length-Length-Variable (3 digit length)
)

// Standard MTI constants
const (
	MTI_AUTH_REQ       = "0200"
	MTI_AUTH_RESP      = "0210"
	MTI_REVERSAL_REQ   = "0400"
	MTI_REVERSAL_RESP  = "0410"
	MTI_RECONCILE_REQ  = "0500"
	MTI_RECONCILE_RESP = "0510"
	MTI_ADMIN_REQ      = "0600"
	MTI_ADMIN_RESP     = "0610"
	MTI_NMM_REQ        = "0800"
	MTI_NMM_RESP       = "0810"
	MTI_ECHO_REQ       = "0800"
	MTI_ECHO_RESP      = "0810"
)

// Response codes
const (
	RC_APPROVED            = "00"
	RC_REFER_TO_ISSUER     = "01"
	RC_INVALID_MERCHANT    = "03"
	RC_DO_NOT_HONOR        = "05"
	RC_INVALID_TRANSACTION = "12"
	RC_INVALID_AMOUNT      = "13"
	RC_INVALID_CARD        = "14"
	RC_FORMAT_ERROR        = "30"
	RC_SYSTEM_ERROR        = "96"
)

// Packager defines field specifications for ISO8583 message format
type Packager struct {
	fields [MaxFields + 1]int16 // Field specifications (0 unused, 1-128)
}

// NewStandardPackager creates a packager with standard ISO8583 field definitions
func NewStandardPackager() Packager {
	var p Packager

	// Standard ISO8583 field definitions
	p.fields[1] = 8        // Bitmap (handled specially)
	p.fields[2] = LLVAR    // Primary Account Number
	p.fields[3] = 6        // Processing Code
	p.fields[4] = 12       // Transaction Amount
	p.fields[5] = 12       // Settlement Amount
	p.fields[6] = 12       // Cardholder Billing Amount
	p.fields[7] = 10       // Transmission Date/Time
	p.fields[8] = 8        // Cardholder Billing Fee
	p.fields[9] = 8        // Settlement Conversion Rate
	p.fields[10] = 8       // Cardholder Billing Conversion Rate
	p.fields[11] = 6       // System Trace Audit Number
	p.fields[12] = 6       // Local Transaction Time
	p.fields[13] = 4       // Local Transaction Date
	p.fields[14] = 4       // Expiration Date
	p.fields[15] = 4       // Settlement Date
	p.fields[18] = 4       // Merchant Category Code
	p.fields[19] = 3       // Acquiring Institution Country Code
	p.fields[22] = 3       // Point of Service Entry Mode
	p.fields[23] = 3       // Application PAN Sequence Number
	p.fields[25] = 2       // Point of Service Condition Code
	p.fields[26] = 2       // Point of Service Capture Code
	p.fields[28] = 8       // Transaction Fee Amount
	p.fields[30] = 8       // Settlement Fee Amount
	p.fields[32] = LLVAR   // Acquiring Institution ID
	p.fields[33] = LLVAR   // Forwarding Institution ID
	p.fields[35] = LLVAR   // Track 2 Data
	p.fields[37] = 12      // Retrieval Reference Number
	p.fields[38] = 6       // Authorization ID Response
	p.fields[39] = 2       // Response Code
	p.fields[40] = 3       // Service Restriction Code
	p.fields[41] = 8       // Card Acceptor Terminal ID
	p.fields[42] = 15      // Card Acceptor ID Code
	p.fields[43] = 40      // Card Acceptor Name/Location
	p.fields[44] = LLVAR   // Additional Response Data
	p.fields[45] = LLVAR   // Track 1 Data
	p.fields[48] = LLLVAR  // Additional Data
	p.fields[49] = 3       // Transaction Currency Code
	p.fields[50] = 3       // Settlement Currency Code
	p.fields[51] = 3       // Cardholder Billing Currency Code
	p.fields[52] = 8       // Personal ID Number Data
	p.fields[53] = 16      // Security Related Control Information
	p.fields[54] = LLLVAR  // Additional Amounts
	p.fields[55] = LLLVAR  // ICC Data
	p.fields[56] = LLLVAR  // Original Data Elements
	p.fields[57] = LLLVAR  // Authorization Life Cycle Code
	p.fields[58] = LLLVAR  // Authorizing Agent Institution
	p.fields[59] = LLLVAR  // Transport Data
	p.fields[60] = LLLVAR  // Reserved National
	p.fields[61] = LLLVAR  // Reserved Private
	p.fields[62] = LLLVAR  // Reserved Private
	p.fields[63] = LLLVAR  // Reserved Private
	p.fields[90] = 42      // Original Data Elements
	p.fields[95] = 42      // Replacement Amounts
	p.fields[102] = LLVAR  // Account ID 1
	p.fields[103] = LLVAR  // Account ID 2
	p.fields[120] = LLLVAR // Reserved Private
	p.fields[121] = LLLVAR // Reserved Private
	p.fields[122] = LLLVAR // Reserved Private
	p.fields[123] = LLLVAR // Reserved Private
	p.fields[124] = LLLVAR // Reserved Private
	p.fields[125] = LLLVAR // Reserved Private
	p.fields[126] = LLLVAR // Reserved Private
	p.fields[127] = LLLVAR // Reserved Private
	p.fields[128] = 8      // Message Authentication Code

	return p
}

// GetFieldSpec returns the field specification
func (p *Packager) GetFieldSpec(fieldNum int) int16 {
	if fieldNum < 1 || fieldNum > MaxFields {
		return 0
	}
	return p.fields[fieldNum]
}

// SetFieldSpec sets a field specification
func (p *Packager) SetFieldSpec(fieldNum int, spec int16) error {
	if fieldNum < 1 || fieldNum > MaxFields {
		return ErrInvalidBit
	}
	p.fields[fieldNum] = spec
	return nil
}

// Field represents a message field with embedded storage
type Field struct {
	data [256]byte // Embedded storage for most fields
	len  int       // Actual data length
	ext  []byte    // Extended storage for larger fields
}

// Bytes returns field data as byte slice (zero-copy)
func (f *Field) Bytes() []byte {
	if f.len == 0 {
		return nil
	}
	if f.ext != nil {
		return f.ext[:f.len]
	}
	return f.data[:f.len]
}

// String returns field data as string (zero-copy)
func (f *Field) String() string {
	if f.len == 0 {
		return ""
	}
	if f.ext != nil {
		return unsafe.String(&f.ext[0], f.len)
	}
	return unsafe.String(&f.data[0], f.len)
}

// Len returns the field data length
func (f *Field) Len() int {
	return f.len
}

// IsEmpty returns true if field has no data
func (f *Field) IsEmpty() bool {
	return f.len == 0
}

// Set sets field data, using embedded storage when possible
func (f *Field) Set(data []byte) {
	dataLen := len(data)
	if dataLen <= len(f.data) {
		// Use embedded storage
		f.len = copy(f.data[:], data)
		f.ext = nil
	} else {
		// Need extended storage
		if len(f.ext) < dataLen {
			f.ext = make([]byte, dataLen)
		}
		f.len = copy(f.ext, data)
	}
}

// SetString sets field data from string
func (f *Field) SetString(s string) {
	f.Set([]byte(s))
}

// Clear clears field data without deallocating extended storage
func (f *Field) Clear() {
	f.len = 0
	// Keep f.ext allocated for reuse
}

// Message represents an ISO8583 message with zero-allocation design
type Message struct {
	mti     [MaxMTILen]byte      // MTI storage
	mtiLen  int                  // MTI length
	fields  [MaxFields + 1]Field // Fields 1-128 (0 unused)
	bitmap  [2]uint64            // Primary and secondary bitmaps
	active  [MaxFields]int       // Active field numbers buffer
	activeN int                  // Number of active fields
	hasBit1 bool                 // Has secondary bitmap
}

// Init initializes the message for reuse (zero allocation)
func (m *Message) Init() {
	m.mtiLen = 0
	m.bitmap[0] = 0
	m.bitmap[1] = 0
	m.hasBit1 = false
	m.activeN = 0

	for i := range m.fields {
		m.fields[i].Clear()
	}
}

// MTI returns MTI as string (zero-copy)
func (m *Message) MTI() string {
	if m.mtiLen == 0 {
		return ""
	}
	return unsafe.String(&m.mti[0], m.mtiLen)
}

// MTIBytes returns MTI as byte slice
func (m *Message) MTIBytes() []byte {
	if m.mtiLen == 0 {
		return nil
	}
	return m.mti[:m.mtiLen]
}

// SetMTI sets MTI from byte slice
func (m *Message) SetMTI(mti []byte) error {
	if len(mti) > MaxMTILen {
		return ErrInvalidLength
	}
	m.mtiLen = copy(m.mti[:], mti)
	return nil
}

// SetMTIString sets MTI from string
func (m *Message) SetMTIString(mti string) error {
	return m.SetMTI([]byte(mti))
}

// SetField sets field data
func (m *Message) SetField(fieldNum int, data []byte) error {
	if fieldNum < 1 || fieldNum > MaxFields {
		return ErrInvalidBit
	}

	m.fields[fieldNum].Set(data)
	m.setBit(fieldNum)
	return nil
}

// SetFieldString sets field from string
func (m *Message) SetFieldString(fieldNum int, data string) error {
	return m.SetField(fieldNum, []byte(data))
}

// GetField returns field data (zero-copy)
func (m *Message) GetField(fieldNum int) ([]byte, error) {
	if fieldNum < 1 || fieldNum > MaxFields {
		return nil, ErrInvalidBit
	}

	field := &m.fields[fieldNum]
	if field.IsEmpty() {
		return nil, ErrFieldNotFound
	}

	return field.Bytes(), nil
}

// GetFieldString returns field as string
func (m *Message) GetFieldString(fieldNum int) (string, error) {
	if fieldNum < 1 || fieldNum > MaxFields {
		return "", ErrInvalidBit
	}

	field := &m.fields[fieldNum]
	if field.IsEmpty() {
		return "", ErrFieldNotFound
	}

	return field.String(), nil
}

// HasField checks if field exists
func (m *Message) HasField(fieldNum int) bool {
	if fieldNum < 1 || fieldNum > MaxFields {
		return false
	}
	return !m.fields[fieldNum].IsEmpty()
}

// RemoveField removes a field
func (m *Message) RemoveField(fieldNum int) error {
	if fieldNum < 1 || fieldNum > MaxFields {
		return ErrInvalidBit
	}

	m.fields[fieldNum].Clear()
	m.clearBit(fieldNum)
	return nil
}

// setBit sets bit in bitmap
func (m *Message) setBit(bitNum int) {
	if bitNum <= 64 {
		m.bitmap[0] |= 1 << (64 - bitNum)
	} else {
		m.bitmap[1] |= 1 << (128 - bitNum)
		m.bitmap[0] |= 1 << 63 // Set bit 1
		m.hasBit1 = true
	}
}

// clearBit clears bit in bitmap
func (m *Message) clearBit(bitNum int) {
	if bitNum <= 64 {
		m.bitmap[0] &^= 1 << (64 - bitNum)
	} else {
		m.bitmap[1] &^= 1 << (128 - bitNum)
	}

	// Check if secondary bitmap still needed
	if m.bitmap[1] == 0 {
		m.bitmap[0] &^= 1 << 63
		m.hasBit1 = false
	}
}

// GetActiveFields returns active field numbers (reuses internal buffer)
func (m *Message) GetActiveFields() []int {
	m.activeN = 0

	// Primary bitmap (skip bit 1 which is for secondary bitmap)
	for i := 2; i <= 64; i++ {
		if m.bitmap[0]&(1<<(64-i)) != 0 {
			m.active[m.activeN] = i
			m.activeN++
		}
	}

	// Secondary bitmap
	if m.hasBit1 && m.bitmap[1] != 0 {
		for i := 65; i <= 128; i++ {
			if m.bitmap[1]&(1<<(128-i)) != 0 {
				m.active[m.activeN] = i
				m.activeN++
			}
		}
	}

	return m.active[:m.activeN]
}

// updateBitmap rebuilds bitmap from field data
func (m *Message) updateBitmap() {
	m.bitmap[0] = 0
	m.bitmap[1] = 0
	m.hasBit1 = false

	for i := 2; i <= MaxFields; i++ {
		if !m.fields[i].IsEmpty() {
			if i <= 64 {
				m.bitmap[0] |= 1 << (64 - i)
			} else {
				m.bitmap[1] |= 1 << (128 - i)
				m.hasBit1 = true
			}
		}
	}

	if m.hasBit1 {
		m.bitmap[0] |= 1 << 63
	}
}

// Pack packs message into provided buffer
func (m *Message) Pack(packager *Packager, buf []byte) (int, error) {
	if len(buf) < MaxMTILen+BitmapLen {
		return 0, ErrBufferTooSmall
	}

	if m.mtiLen == 0 {
		return 0, ErrInvalidMTI
	}

	pos := 0

	// Write MTI
	pos += copy(buf[pos:], m.mti[:m.mtiLen])

	// Update bitmap
	m.updateBitmap()

	// Write primary bitmap
	pos += writeBitmapHex(m.bitmap[0], buf[pos:])

	// Write secondary bitmap if needed
	if m.hasBit1 {
		pos += writeBitmapHex(m.bitmap[1], buf[pos:])
	}

	// Pack fields
	activeFields := m.GetActiveFields()
	for i := 0; i < m.activeN; i++ {
		fieldNum := activeFields[i]
		fieldData := m.fields[fieldNum].Bytes()
		if fieldData == nil {
			continue
		}

		spec := packager.GetFieldSpec(fieldNum)
		written, err := packField(spec, fieldData, buf[pos:])
		if err != nil {
			return 0, err
		}
		pos += written
	}

	return pos, nil
}

// packField packs a single field
func packField(spec int16, data []byte, buf []byte) (int, error) {
	dataLen := len(data)
	pos := 0

	switch {
	case spec > 0:
		// Fixed length
		requiredLen := int(spec)
		if len(buf) < requiredLen {
			return 0, ErrBufferTooSmall
		}

		if dataLen > requiredLen {
			pos += copy(buf[pos:], data[:requiredLen])
		} else {
			pos += copy(buf[pos:], data)
			for i := dataLen; i < requiredLen; i++ {
				buf[pos] = ' '
				pos++
			}
		}

	case spec == LLVAR:
		if dataLen > 99 {
			return 0, ErrInvalidLength
		}
		if len(buf) < 2+dataLen {
			return 0, ErrBufferTooSmall
		}

		buf[pos] = byte('0' + dataLen/10)
		buf[pos+1] = byte('0' + dataLen%10)
		pos += 2
		pos += copy(buf[pos:], data)

	case spec == LLLVAR:
		if dataLen > 999 {
			return 0, ErrInvalidLength
		}
		if len(buf) < 3+dataLen {
			return 0, ErrBufferTooSmall
		}

		buf[pos] = byte('0' + dataLen/100)
		buf[pos+1] = byte('0' + (dataLen/10)%10)
		buf[pos+2] = byte('0' + dataLen%10)
		pos += 3
		pos += copy(buf[pos:], data)

	default:
		return 0, ErrInvalidPackager
	}

	return pos, nil
}

// Parse parses message from byte data
func (m *Message) Parse(packager *Packager, data []byte) error {
	if len(data) < MaxMTILen {
		return ErrInsufficientData
	}

	m.Init()
	pos := 0

	// Parse MTI
	m.mtiLen = MaxMTILen
	copy(m.mti[:], data[pos:pos+MaxMTILen])
	pos += MaxMTILen

	if !isValidMTI(m.mti[:m.mtiLen]) {
		return ErrInvalidMTI
	}

	// Parse primary bitmap
	if len(data) < pos+BitmapLen {
		return ErrInsufficientData
	}

	bitmap, err := parseHexUint64(data[pos : pos+BitmapLen])
	if err != nil {
		return ErrInvalidBitmap
	}
	m.bitmap[0] = bitmap
	pos += BitmapLen

	// Parse secondary bitmap if present
	m.hasBit1 = (bitmap & (1 << 63)) != 0
	if m.hasBit1 {
		if len(data) < pos+BitmapLen {
			return ErrInsufficientData
		}

		m.bitmap[1], err = parseHexUint64(data[pos : pos+BitmapLen])
		if err != nil {
			return ErrInvalidBitmap
		}
		pos += BitmapLen
	}

	// Parse fields
	activeFields := m.GetActiveFields()
	for i := 0; i < m.activeN; i++ {
		fieldNum := activeFields[i]
		spec := packager.GetFieldSpec(fieldNum)

		consumed, err := parseField(&m.fields[fieldNum], spec, data[pos:])
		if err != nil {
			return err
		}
		pos += consumed
	}

	return nil
}

// parseField parses a single field
func parseField(field *Field, spec int16, data []byte) (int, error) {
	pos := 0

	switch {
	case spec > 0:
		// Fixed length
		requiredLen := int(spec)
		if len(data) < requiredLen {
			return 0, ErrInsufficientData
		}

		field.Set(data[:requiredLen])
		pos += requiredLen

	case spec == LLVAR:
		if len(data) < 2 {
			return 0, ErrInsufficientData
		}

		length, err := parseDecimal2(data[pos : pos+2])
		if err != nil {
			return 0, err
		}
		pos += 2

		if len(data) < pos+length {
			return 0, ErrInsufficientData
		}

		if length > 0 {
			field.Set(data[pos : pos+length])
		}
		pos += length

	case spec == LLLVAR:
		if len(data) < 3 {
			return 0, ErrInsufficientData
		}

		length, err := parseDecimal3(data[pos : pos+3])
		if err != nil {
			return 0, err
		}
		pos += 3

		if len(data) < pos+length {
			return 0, ErrInsufficientData
		}

		if length > 0 {
			field.Set(data[pos : pos+length])
		}
		pos += length

	default:
		return 0, ErrInvalidPackager
	}

	return pos, nil
}

// CopyRequiredFields copies standard echo fields from source to destination
func (m *Message) CopyRequiredFields(src *Message) {
	echoFields := []int{2, 3, 4, 7, 11, 12, 13, 22, 32, 37, 41, 42, 49}
	for _, fieldNum := range echoFields {
		if data, err := src.GetField(fieldNum); err == nil {
			m.SetField(fieldNum, data)
		}
	}
}

// Utility functions

// writeBitmapHex writes uint64 as hex to buffer
func writeBitmapHex(bitmap uint64, buf []byte) int {
	const hexChars = "0123456789ABCDEF"
	if len(buf) < BitmapLen {
		return 0
	}

	for i := 15; i >= 0; i-- {
		buf[15-i] = hexChars[(bitmap>>(i*4))&0xF]
	}
	return BitmapLen
}

// parseHexUint64 parses hex bytes to uint64
func parseHexUint64(hexBytes []byte) (uint64, error) {
	if len(hexBytes) != BitmapLen {
		return 0, ErrInvalidLength
	}

	var result uint64
	for _, b := range hexBytes {
		result <<= 4
		switch {
		case b >= '0' && b <= '9':
			result |= uint64(b - '0')
		case b >= 'A' && b <= 'F':
			result |= uint64(b - 'A' + 10)
		case b >= 'a' && b <= 'f':
			result |= uint64(b - 'a' + 10)
		default:
			return 0, ErrInvalidLength
		}
	}
	return result, nil
}

// parseDecimal2 parses 2-digit decimal
func parseDecimal2(b []byte) (int, error) {
	if len(b) != 2 || b[0] < '0' || b[0] > '9' || b[1] < '0' || b[1] > '9' {
		return 0, ErrInvalidLength
	}
	return int(b[0]-'0')*10 + int(b[1]-'0'), nil
}

// parseDecimal3 parses 3-digit decimal
func parseDecimal3(b []byte) (int, error) {
	if len(b) != 3 || b[0] < '0' || b[0] > '9' || b[1] < '0' || b[1] > '9' || b[2] < '0' || b[2] > '9' {
		return 0, ErrInvalidLength
	}
	return int(b[0]-'0')*100 + int(b[1]-'0')*10 + int(b[2]-'0'), nil
}

// isValidMTI validates MTI format
func isValidMTI(mti []byte) bool {
	if len(mti) != 4 {
		return false
	}
	for _, b := range mti {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

// IsRequest checks if MTI is request
func IsRequest(mti []byte) bool {
	return len(mti) == 4 && mti[2] == '0' && mti[3] == '0'
}

// IsResponse checks if MTI is response
func IsResponse(mti []byte) bool {
	return len(mti) == 4 && mti[2] == '1' && mti[3] == '0'
}

// GetResponseMTI converts request MTI to response
func GetResponseMTI(requestMTI []byte, responseMTI []byte) error {
	if len(requestMTI) != 4 || len(responseMTI) < 4 {
		return ErrInvalidMTI
	}
	if !IsRequest(requestMTI) {
		return ErrInvalidMTI
	}
	copy(responseMTI, requestMTI)
	responseMTI[2] = '1'
	responseMTI[3] = '0'
	return nil
}

// Example usage without pools
func ExampleUsage() {
	packager := NewStandardPackager()

	// Stack-allocated message - zero heap allocation
	var msg Message

	// Parse incoming data
	data := []byte("0200...")
	if err := msg.Parse(&packager, data); err != nil {
		return
	}

	// Create response message
	var response Message
	response.SetMTIString(MTI_AUTH_RESP)
	response.CopyRequiredFields(&msg)
	response.SetFieldString(39, RC_APPROVED)

	// Pack response
	buf := make([]byte, 4096)
	length, err := response.Pack(&packager, buf)
	if err != nil {
		return
	}

	// Send buf[:length]
	_ = buf[:length]
}
