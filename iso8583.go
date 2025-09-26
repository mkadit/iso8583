package iso8583

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Core error types
var (
	ErrInvalidLength     = errors.New("invalid length")
	ErrInsufficientData  = errors.New("insufficient data")
	ErrInvalidMTI        = errors.New("invalid MTI")
	ErrInvalidBit        = errors.New("invalid bit")
	ErrInvalidPackager   = errors.New("invalid packager")
	ErrBufferTooSmall    = errors.New("buffer too small")
	ErrFieldNotFound     = errors.New("field not found")
	ErrInvalidBitmap     = errors.New("invalid bitmap")
	ErrInvalidTLV        = errors.New("invalid TLV structure")
	ErrUnsupportedFormat = errors.New("unsupported format")
	ErrInvalidTag        = errors.New("invalid tag")
	ErrHexDataOddLength  = errors.New("hex data must have even length")
)

// HeaderType defines how ISO8583 message length headers are encoded
type HeaderType int

const (
	HeaderNone   HeaderType = iota
	HeaderBinary            // 2-byte binary length
	HeaderASCII             // 4-digit ASCII decimal length
	HeaderHex               // 4-char ASCII hex length
	HeaderCustom            // Custom header format
)

// HeaderConfig defines configuration for custom headers
type HeaderConfig struct {
	Type   HeaderType
	Length int // Header length in bytes for binary, or char count for ASCII/Hex
}

// DefaultHeaderConfigs provides standard header configurations
var DefaultHeaderConfigs = map[HeaderType]HeaderConfig{
	HeaderNone:   {Type: HeaderNone, Length: 0},
	HeaderBinary: {Type: HeaderBinary, Length: 2},
	HeaderASCII:  {Type: HeaderASCII, Length: 4},
	HeaderHex:    {Type: HeaderHex, Length: 4},
}

// Packager defines the message structure and field formats
type Packager struct {
	IsoPackager []int // Field definitions (0 = not used, positive = fixed length, negative = variable)
}

// DefaultISOPackager returns a standard ISO8583 packager configuration
func DefaultISOPackager() *Packager {
	p := &Packager{
		IsoPackager: make([]int, 129), // 0-128 (field 0 is MTI)
	}

	// Standard ISO8583 field definitions
	p.IsoPackager[0] = 4     // MTI (fixed 4 digits)
	p.IsoPackager[1] = 64    // Primary bitmap (fixed 64 bits = 16 hex chars)
	p.IsoPackager[2] = -19   // Primary account number (variable 2-19 digits)
	p.IsoPackager[3] = 6     // Processing code (fixed 6 digits)
	p.IsoPackager[4] = 12    // Transaction amount (fixed 12 digits)
	p.IsoPackager[7] = 10    // Transmission date & time (fixed 10 digits)
	p.IsoPackager[11] = 6    // System trace audit number (fixed 6 digits)
	p.IsoPackager[12] = 6    // Local transaction time (fixed 6 digits)
	p.IsoPackager[13] = 4    // Local transaction date (fixed 4 digits)
	p.IsoPackager[22] = 3    // POS entry mode (fixed 3 digits)
	p.IsoPackager[25] = 2    // POS condition code (fixed 2 digits)
	p.IsoPackager[35] = -37  // Track 2 data (variable 2-37 digits)
	p.IsoPackager[37] = 12   // Retrieval reference number (fixed 12 digits)
	p.IsoPackager[38] = 6    // Authorization ID response (fixed 6 digits)
	p.IsoPackager[39] = 2    // Response code (fixed 2 digits)
	p.IsoPackager[41] = 8    // Card acceptor terminal ID (fixed 8 digits)
	p.IsoPackager[42] = 15   // Card acceptor ID code (fixed 15 digits)
	p.IsoPackager[48] = -999 // Additional data (variable)
	p.IsoPackager[55] = -999 // EMV data (variable)
	p.IsoPackager[70] = 3    // Network management info code (fixed 3 digits)

	return p
}

// Message represents an ISO8583 message
type Message struct {
	MTI         []byte
	Bitmap      [129]bool   // Index 0 unused, 1-128 for fields
	Fields      [129][]byte // Index 0 unused, 1-128 for fields
	Packager    *Packager
	FullMessage []byte
}

// NewMessage creates a new ISO8583 message
func NewMessage(packager *Packager) *Message {
	if packager == nil {
		packager = DefaultISOPackager()
	}
	return &Message{
		Packager: packager,
	}
}

// Parse parses message from byte data.
// IMPORTANT: This is a zero-allocation parse. The fields in the resulting Message
// will be slices that point to the original `data` byte array. The caller MUST
// ensure the `data` array remains valid for the lifetime of the Message.
func (m *Message) Parse(data []byte) error {
	if len(data) < 4 {
		return ErrInvalidMTI
	}
	// Reset bitmap for reparsing
	for i := range m.Bitmap {
		m.Bitmap[i] = false
	}

	// Parse MTI (zero-copy)
	m.MTI = data[:4]
	pos := 4

	// Parse primary bitmap
	if len(data) < pos+16 {
		return ErrInvalidBitmap
	}

	var bitmap1 [8]byte
	_, err := hex.Decode(bitmap1[:], data[pos:pos+16])
	if err != nil {
		return fmt.Errorf("invalid primary bitmap: %w", err)
	}
	pos += 16

	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			bit := i*8 + j + 1
			if bitmap1[i]&(0x80>>j) != 0 {
				m.Bitmap[bit] = true
			}
		}
	}

	// Check for secondary bitmap
	if m.Bitmap[1] {
		if len(data) < pos+16 {
			return ErrInvalidBitmap
		}

		var bitmap2 [8]byte
		_, err := hex.Decode(bitmap2[:], data[pos:pos+16])
		if err != nil {
			return fmt.Errorf("invalid secondary bitmap: %w", err)
		}
		pos += 16

		for i := 0; i < 8; i++ {
			for j := 0; j < 8; j++ {
				bit := i*8 + j + 65
				if bitmap2[i]&(0x80>>j) != 0 {
					m.Bitmap[bit] = true
				}
			}
		}
	}

	// Parse fields
	for fieldNum := 2; fieldNum <= 128; fieldNum++ {
		if !m.Bitmap[fieldNum] {
			continue
		}

		fieldDef := m.Packager.IsoPackager[fieldNum]
		if fieldDef == 0 {
			continue
		}

		if fieldDef > 0 { // Fixed length
			if len(data) < pos+fieldDef {
				return fmt.Errorf("insufficient data for field %d", fieldNum)
			}
			m.Fields[fieldNum] = data[pos : pos+fieldDef] // Zero-copy slice
			pos += fieldDef
		} else { // Variable length
			maxLen := -fieldDef
			var lenDigits int
			if maxLen <= 99 {
				lenDigits = 2
			} else if maxLen <= 999 {
				lenDigits = 3
			} else {
				lenDigits = 4
			}

			if len(data) < pos+lenDigits {
				return fmt.Errorf("insufficient data for length of field %d", fieldNum)
			}

			fieldLen, err := parseASCIIToInt(data[pos : pos+lenDigits])
			if err != nil {
				return fmt.Errorf("invalid length for field %d: %w", fieldNum, err)
			}
			pos += lenDigits

			if len(data) < pos+fieldLen {
				return fmt.Errorf("insufficient data for value of field %d", fieldNum)
			}

			m.Fields[fieldNum] = data[pos : pos+fieldLen] // Zero-copy slice
			pos += fieldLen
		}
	}

	return nil
}

// Pack packs message into provided buffer
func (m *Message) Pack(buf []byte) (int, error) {
	if len(buf) < 4 {
		return 0, ErrBufferTooSmall
	}
	if len(m.MTI) != 4 {
		return 0, ErrInvalidMTI
	}

	copy(buf, m.MTI)
	pos := 4

	needSecondary := false
	for i := 65; i <= 128; i++ {
		if m.Bitmap[i] {
			needSecondary = true
			break
		}
	}

	m.Bitmap[1] = needSecondary

	if len(buf) < pos+16 {
		return 0, ErrBufferTooSmall
	}

	var bitmap1 [8]byte
	for i := 1; i <= 64; i++ {
		if m.Bitmap[i] {
			byteIdx := (i - 1) / 8
			bitIdx := (i - 1) % 8
			bitmap1[byteIdx] |= 0x80 >> bitIdx
		}
	}
	hex.Encode(buf[pos:pos+16], bitmap1[:])
	pos += 16

	if needSecondary {
		if len(buf) < pos+16 {
			return 0, ErrBufferTooSmall
		}
		var bitmap2 [8]byte
		for i := 65; i <= 128; i++ {
			if m.Bitmap[i] {
				byteIdx := (i - 65) / 8
				bitIdx := (i - 65) % 8
				bitmap2[byteIdx] |= 0x80 >> bitIdx
			}
		}
		hex.Encode(buf[pos:pos+16], bitmap2[:])
		pos += 16
	}

	for fieldNum := 2; fieldNum <= 128; fieldNum++ {
		if !m.Bitmap[fieldNum] {
			continue
		}

		fieldDef := m.Packager.IsoPackager[fieldNum]
		if fieldDef == 0 {
			continue
		}
		fieldData := m.Fields[fieldNum]

		if fieldDef > 0 { // Fixed
			if len(buf) < pos+fieldDef {
				return 0, ErrBufferTooSmall
			}
			copy(buf[pos:pos+fieldDef], fieldData)
			if len(fieldData) < fieldDef {
				for i := len(fieldData); i < fieldDef; i++ {
					buf[pos+i] = ' '
				}
			}
			pos += fieldDef
		} else { // Variable
			fieldLen := len(fieldData)
			maxLen := -fieldDef
			var lenDigits int
			if maxLen <= 99 {
				lenDigits = 2
			} else if maxLen <= 999 {
				lenDigits = 3
			} else {
				lenDigits = 4
			}

			if len(buf) < pos+lenDigits+fieldLen {
				return 0, ErrBufferTooSmall
			}

			writeIntToASCII(buf[pos:pos+lenDigits], fieldLen, lenDigits)
			pos += lenDigits

			copy(buf[pos:pos+fieldLen], fieldData)
			pos += fieldLen
		}
	}
	return pos, nil
}

// SetField sets field data by copying it into the message.
func (m *Message) SetField(fieldNum int, data []byte) error {
	if fieldNum < 1 || fieldNum > 128 {
		return fmt.Errorf("invalid field number: %d", fieldNum)
	}
	m.Bitmap[fieldNum] = true
	m.Fields[fieldNum] = make([]byte, len(data)) // Make a copy for safety when building
	copy(m.Fields[fieldNum], data)
	return nil
}

// GetField returns field data
func (m *Message) GetField(fieldNum int) ([]byte, error) {
	if fieldNum < 1 || fieldNum > 128 {
		return nil, fmt.Errorf("invalid field number: %d", fieldNum)
	}
	if !m.Bitmap[fieldNum] {
		return nil, ErrFieldNotFound
	}
	return m.Fields[fieldNum], nil
}

// SetMTI sets the message type indicator by copying from a string.
func (m *Message) SetMTI(mti string) error {
	if len(mti) != 4 {
		return ErrInvalidMTI
	}
	if m.MTI == nil || cap(m.MTI) < 4 {
		m.MTI = make([]byte, 4)
	}
	copy(m.MTI, mti)
	return nil
}

// GetBytes returns the complete message as byte array (without header)
func (m *Message) GetBytes() ([]byte, error) {
	buf := make([]byte, 8192) // Allocate sufficient buffer
	length, err := m.Pack(buf)
	if err != nil {
		return nil, err
	}

	result := make([]byte, length)
	copy(result, buf[:length])
	m.FullMessage = result
	return result, nil
}

// GetBytesWithHeader returns the complete message with header as byte array
func (m *Message) GetBytesWithHeader(headerConfig HeaderConfig) ([]byte, error) {
	buf := make([]byte, 8192) // Allocate sufficient buffer
	length, err := m.PackWithHeader(buf, headerConfig)
	if err != nil {
		return nil, err
	}

	result := make([]byte, length)
	copy(result, buf[:length])
	m.FullMessage = result
	return result, nil
}

// PackWithHeader packs message with header
func (m *Message) PackWithHeader(buf []byte, headerConfig HeaderConfig) (int, error) {
	headerLen := headerConfig.Length
	if headerConfig.Type == HeaderNone {
		headerLen = 0
	}
	if len(buf) < headerLen {
		return 0, ErrBufferTooSmall
	}
	msgLen, err := m.Pack(buf[headerLen:])
	if err != nil {
		return 0, err
	}
	writtenHeaderLen, err := WriteHeader(msgLen, buf, headerConfig)
	if err != nil {
		return 0, err
	}
	return writtenHeaderLen + msgLen, nil
}

// ParseWithHeader parses message with header
func (m *Message) ParseWithHeader(data []byte, headerConfig HeaderConfig) error {
	msgLen, headerLen, err := ReadHeader(data, headerConfig)
	if err != nil {
		return err
	}
	if len(data) < headerLen+msgLen {
		return ErrInsufficientData
	}
	return m.Parse(data[headerLen : headerLen+msgLen])
}

// WriteHeader writes the message length header
func WriteHeader(msgLen int, buf []byte, config HeaderConfig) (int, error) {
	if len(buf) < config.Length {
		return 0, ErrBufferTooSmall
	}
	switch config.Type {
	case HeaderNone:
		return 0, nil
	case HeaderBinary:
		return writeBinaryHeader(msgLen, buf, config)
	case HeaderASCII:
		return writeASCIIHeader(msgLen, buf, config)
	case HeaderHex:
		return writeHexHeader(msgLen, buf, config)
	}
	return 0, errors.New("unsupported header type")
}

// ReadHeader reads the message length header
func ReadHeader(buf []byte, config HeaderConfig) (int, int, error) {
	if len(buf) < config.Length {
		return 0, 0, ErrInsufficientData
	}
	switch config.Type {
	case HeaderNone:
		return len(buf), 0, nil
	case HeaderBinary:
		return readBinaryHeader(buf, config)
	case HeaderASCII:
		return readASCIIHeader(buf, config)
	case HeaderHex:
		return readHexHeader(buf, config)
	}
	return 0, 0, errors.New("unsupported header type")
}

func writeBinaryHeader(msgLen int, buf []byte, config HeaderConfig) (int, error) {
	switch config.Length {
	case 2:
		buf[0] = byte(msgLen >> 8)
		buf[1] = byte(msgLen)
		return 2, nil
	case 4:
		buf[0] = byte(msgLen >> 24)
		buf[1] = byte(msgLen >> 16)
		buf[2] = byte(msgLen >> 8)
		buf[3] = byte(msgLen)
		return 4, nil
	}
	return 0, ErrInvalidLength
}

func readBinaryHeader(buf []byte, config HeaderConfig) (int, int, error) {
	switch config.Length {
	case 2:
		return int(buf[0])<<8 | int(buf[1]), 2, nil
	case 4:
		return int(buf[0])<<24 | int(buf[1])<<16 | int(buf[2])<<8 | int(buf[3]), 4, nil
	}
	return 0, 0, ErrInvalidLength
}

func writeASCIIHeader(msgLen int, buf []byte, config HeaderConfig) (int, error) {
	writeIntToASCII(buf, msgLen, config.Length)
	return config.Length, nil
}

func readASCIIHeader(buf []byte, config HeaderConfig) (int, int, error) {
	msgLen, err := parseASCIIToInt(buf[:config.Length])
	if err != nil {
		return 0, 0, ErrInvalidLength
	}
	return msgLen, config.Length, nil
}

func writeHexHeader(msgLen int, buf []byte, config HeaderConfig) (int, error) {
	s := strconv.FormatInt(int64(msgLen), 16)
	if len(s) > config.Length {
		return 0, ErrInvalidLength
	}
	// Add padding
	if len(s) < config.Length {
		padding := config.Length - len(s)
		for i := 0; i < padding; i++ {
			buf[i] = '0'
		}
		copy(buf[padding:], s)
	} else {
		copy(buf, s)
	}
	return config.Length, nil
}

func readHexHeader(buf []byte, config HeaderConfig) (int, int, error) {
	msgLen, err := strconv.ParseInt(string(buf[:config.Length]), 16, 64)
	if err != nil {
		return 0, 0, ErrInvalidLength
	}
	return int(msgLen), config.Length, nil
}

// TLV Implementation
type TLVConfig struct {
	TagLength    int // Tag length in bytes
	LengthLength int // Length field size in bytes
}
type TLVEntry struct {
	Tag   string
	Value []byte
}
type TLV struct {
	Entries []TLVEntry
}

func NewTLV() *TLV {
	return &TLV{}
}

func (t *TLV) AddEntry(tag string, value []byte) {
	t.Entries = append(t.Entries, TLVEntry{Tag: tag, Value: value})
}

func (t *TLV) GetEntry(tag string) ([]byte, error) {
	for _, entry := range t.Entries {
		if entry.Tag == tag {
			return entry.Value, nil
		}
	}
	return nil, ErrFieldNotFound
}

func (t *TLV) ParseTLV(data []byte, config TLVConfig) error {
	pos := 0
	t.Entries = t.Entries[:0]
	for pos < len(data) {
		if data[pos] == 0x00 {
			break
		}
		if pos+config.TagLength > len(data) {
			return ErrInsufficientData
		}
		tag := string(data[pos : pos+config.TagLength])
		pos += config.TagLength

		if pos+config.LengthLength > len(data) {
			return ErrInsufficientData
		}
		var length int
		var err error
		if config.LengthLength == 1 {
			length = int(data[pos])
		} else if config.LengthLength == 2 {
			length = int(data[pos])<<8 | int(data[pos+1])
		} else {
			length, err = parseASCIIToInt(data[pos : pos+config.LengthLength])
			if err != nil {
				return fmt.Errorf("invalid length: %w", err)
			}
		}
		pos += config.LengthLength

		if pos+length > len(data) {
			return ErrInsufficientData
		}
		value := data[pos : pos+length] // Zero-copy slice
		pos += length
		t.Entries = append(t.Entries, TLVEntry{Tag: tag, Value: value})
	}
	return nil
}

func (t *TLV) BuildTLV(buf []byte, config TLVConfig) (int, error) {
	pos := 0
	for _, entry := range t.Entries {
		required := config.TagLength + config.LengthLength + len(entry.Value)
		if pos+required > len(buf) {
			return 0, ErrBufferTooSmall
		}
		copy(buf[pos:pos+config.TagLength], entry.Tag)
		pos += config.TagLength

		if config.LengthLength == 1 {
			buf[pos] = byte(len(entry.Value))
		} else if config.LengthLength == 2 {
			buf[pos] = byte(len(entry.Value) >> 8)
			buf[pos+1] = byte(len(entry.Value))
		} else {
			writeIntToASCII(buf[pos:pos+config.LengthLength], len(entry.Value), config.LengthLength)
		}
		pos += config.LengthLength

		copy(buf[pos:pos+len(entry.Value)], entry.Value)
		pos += len(entry.Value)
	}
	return pos, nil
}

func (t *TLV) ParseEMVTLV(data []byte) error {
	pos := 0
	t.Entries = t.Entries[:0]
	var tagBuilder strings.Builder

	for pos < len(data) {
		if data[pos] == 0x00 {
			break
		}
		if pos >= len(data) {
			break
		}
		tagBuilder.Reset()

		firstByte := data[pos]
		fmt.Fprintf(&tagBuilder, "%02X", firstByte)
		pos++

		if (firstByte & 0x1F) == 0x1F {
			for pos < len(data) {
				tagByte := data[pos]
				fmt.Fprintf(&tagBuilder, "%02X", tagByte)
				pos++
				if (tagByte & 0x80) == 0 {
					break
				}
			}
		}

		if pos >= len(data) {
			return ErrInsufficientData
		}

		length := 0
		firstLenByte := data[pos]
		pos++
		if (firstLenByte & 0x80) == 0 {
			length = int(firstLenByte)
		} else {
			numLenBytes := int(firstLenByte & 0x7F)
			if numLenBytes == 0 || pos+numLenBytes > len(data) {
				return ErrInvalidLength
			}
			for i := 0; i < numLenBytes; i++ {
				length = (length << 8) | int(data[pos])
				pos++
			}
		}

		if pos+length > len(data) {
			return ErrInsufficientData
		}
		value := data[pos : pos+length]
		pos += length
		t.Entries = append(t.Entries, TLVEntry{Tag: tagBuilder.String(), Value: value})
	}
	return nil
}

func (t *TLV) BuildEMVTLV(buf []byte) (int, error) {
	pos := 0
	for _, entry := range t.Entries {
		tagBytes, err := hex.DecodeString(entry.Tag)
		if err != nil {
			return 0, fmt.Errorf("invalid tag %s: %w", entry.Tag, err)
		}
		if pos+len(tagBytes) > len(buf) {
			return 0, ErrBufferTooSmall
		}
		copy(buf[pos:], tagBytes)
		pos += len(tagBytes)

		length := len(entry.Value)
		if length < 0x80 {
			if pos+1 > len(buf) {
				return 0, ErrBufferTooSmall
			}
			buf[pos] = byte(length)
			pos++
		} else {
			var lenBytes []byte
			l := length
			for l > 0 {
				lenBytes = append([]byte{byte(l)}, lenBytes...)
				l >>= 8
			}
			if pos+1+len(lenBytes) > len(buf) {
				return 0, ErrBufferTooSmall
			}
			buf[pos] = byte(0x80 | len(lenBytes))
			pos++
			copy(buf[pos:], lenBytes)
			pos += len(lenBytes)
		}

		if pos+length > len(buf) {
			return 0, ErrBufferTooSmall
		}
		copy(buf[pos:], entry.Value)
		pos += length
	}
	return pos, nil
}

// --- Performance Helpers ---
func writeIntToASCII(buf []byte, val, digits int) {
	for i := digits - 1; i >= 0; i-- {
		buf[i] = byte(val%10 + '0')
		val /= 10
	}
}

func parseASCIIToInt(b []byte) (int, error) {
	n := 0
	for _, ch := range b {
		ch -= '0'
		if ch > 9 {
			return 0, errors.New("invalid character in numeric string")
		}
		n = n*10 + int(ch)
	}
	return n, nil
}
