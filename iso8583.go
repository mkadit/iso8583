package iso8583

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Core error types
var (
	ErrInvalidLength         = errors.New("invalid length")
	ErrInsufficientData      = errors.New("insufficient data")
	ErrInvalidMTI            = errors.New("invalid MTI")
	ErrInvalidBit            = errors.New("invalid bit")
	ErrInvalidPackager       = errors.New("invalid packager")
	ErrBufferTooSmall        = errors.New("buffer too small")
	ErrFieldNotFound         = errors.New("field not found")
	ErrInvalidBitmap         = errors.New("invalid bitmap")
	ErrInvalidTLV            = errors.New("invalid TLV structure")
	ErrUnsupportedFormat     = errors.New("unsupported format")
	ErrInvalidTag            = errors.New("invalid tag")
	ErrHexDataOddLength      = errors.New("hex data must have even length")
	ErrMissingMandatoryField = errors.New("missing mandatory field")
)

// HeaderType defines how ISO8583 message length headers are encoded
type HeaderType int

const (
	HeaderNone   HeaderType = iota
	HeaderBinary            // 2-byte binary length
	HeaderASCII             // 4-digit ASCII decimal length
	HeaderHex               // 4-char ASCII hex length
)

// FieldType defines the data type and encoding of a field.
type FieldType string

const (
	ANS    FieldType = "ans" // Alphanumeric, special characters
	N      FieldType = "n"   // Numeric
	B      FieldType = "b"   // Binary
	Z      FieldType = "z"   // Tracks 2 and 3 code set
	Custom FieldType = "custom"
)

// LengthType defines how the length of a field is determined.
type LengthType string

const (
	FIXED  LengthType = "FIXED"
	LLVAR  LengthType = "LLVAR"
	LLLVAR LengthType = "LLLVAR"
)

// FieldDefinition describes a single field in an ISO8583 message.
type FieldDefinition struct {
	Type        FieldType
	LengthType  LengthType
	MaxLength   int
	IsMandatory bool // For validation purposes, not packing/parsing logic
}

// Packager holds the complete definition for an ISO8583 message format.
type Packager struct {
	Fields [129]FieldDefinition
}

// A temporary struct matching the JSON structure for easy unmarshalling.
type jsonFieldDefinition struct {
	IsMandatory bool `json:"isMandatory"`
	Type        FieldType
	Length      struct {
		Type LengthType
		Max  int
	}
}

// NewPackagerFromJSON creates a new Packager from a JSON configuration.
func NewPackagerFromJSON(configData []byte) (*Packager, error) {
	var config map[string]jsonFieldDefinition
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal packager JSON: %w", err)
	}

	packager := &Packager{}
	for fieldStr, fieldConfig := range config {
		fieldNum, err := strconv.Atoi(fieldStr)
		if err != nil {
			return nil, fmt.Errorf("invalid field number in JSON: %s", fieldStr)
		}
		if fieldNum < 1 || fieldNum > 128 {
			return nil, fmt.Errorf("field number out of range (1-128): %d", fieldNum)
		}

		packager.Fields[fieldNum] = FieldDefinition{
			IsMandatory: fieldConfig.IsMandatory,
			Type:        fieldConfig.Type,
			LengthType:  fieldConfig.Length.Type,
			MaxLength:   fieldConfig.Length.Max,
		}
	}
	return packager, nil
}

// DefaultPackager returns a packager based on the ISO 8583-1:1987 standard.
func DefaultPackager() *Packager {
	p := &Packager{}
	p.Fields[1] = FieldDefinition{Type: B, LengthType: FIXED, MaxLength: 16, IsMandatory: true} // Bitmap
	p.Fields[2] = FieldDefinition{Type: N, LengthType: LLVAR, MaxLength: 19}                    // Primary account number (PAN)
	p.Fields[3] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 6, IsMandatory: true}  // Processing code
	p.Fields[4] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 12}                    // Amount, transaction
	p.Fields[5] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 12}                    // Amount, settlement
	p.Fields[6] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 12}                    // Amount, cardholder billing
	p.Fields[7] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 10, IsMandatory: true} // Transmission date & time
	p.Fields[8] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 8}                     // Amount, cardholder billing fee
	p.Fields[9] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 8}                     // Conversion rate, settlement
	p.Fields[10] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 8}                    // Conversion rate, cardholder billing
	p.Fields[11] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 6, IsMandatory: true} // System trace audit number (STAN)
	p.Fields[12] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 6}                    // Time, local transaction
	p.Fields[13] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 4}                    // Date, local transaction
	p.Fields[14] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 4}                    // Date, expiration
	p.Fields[15] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 4}                    // Date, settlement
	p.Fields[16] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 4}                    // Date, conversion
	p.Fields[17] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 4}                    // Date, capture
	p.Fields[18] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 4}                    // Merchant type
	p.Fields[19] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 3}                    // Acquiring institution country code
	p.Fields[20] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 3}                    // PAN extended, country code
	p.Fields[21] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 3}                    // Forwarding institution. country code
	p.Fields[22] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 3}                    // Point of service entry mode
	p.Fields[23] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 3}                    // Application PAN sequence number
	p.Fields[24] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 3}                    // Network International identifier (NII)
	p.Fields[25] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 2}                    // Point of service condition code
	p.Fields[26] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 2}                    // Point of service capture code
	p.Fields[27] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 1}                    // Authorizing identification response length
	p.Fields[28] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 9}                    // Amount, transaction fee
	p.Fields[29] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 9}                    // Amount, settlement fee
	p.Fields[30] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 9}                    // Amount, transaction processing fee
	p.Fields[31] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 9}                    // Amount, settlement processing fee
	p.Fields[32] = FieldDefinition{Type: N, LengthType: LLVAR, MaxLength: 11}                   // Acquiring institution identification code
	p.Fields[33] = FieldDefinition{Type: N, LengthType: LLVAR, MaxLength: 11}                   // Forwarding institution identification code
	p.Fields[34] = FieldDefinition{Type: ANS, LengthType: LLVAR, MaxLength: 28}                 // Primary account number, extended
	p.Fields[35] = FieldDefinition{Type: Z, LengthType: LLVAR, MaxLength: 37}                   // Track 2 data
	p.Fields[36] = FieldDefinition{Type: N, LengthType: LLLVAR, MaxLength: 104}                 // Track 3 data
	p.Fields[37] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 12}                 // Retrieval reference number
	p.Fields[38] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 6}                  // Authorization identification response
	p.Fields[39] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 2}                  // Response code
	p.Fields[40] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 3}                  // Service restriction code
	p.Fields[41] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 8}                  // Card acceptor terminal identification
	p.Fields[42] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 15}                 // Card acceptor identification code
	p.Fields[43] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 40}                 // Card acceptor name/location
	p.Fields[44] = FieldDefinition{Type: ANS, LengthType: LLVAR, MaxLength: 25}                 // Additional response data
	p.Fields[45] = FieldDefinition{Type: ANS, LengthType: LLVAR, MaxLength: 76}                 // Track 1 data
	p.Fields[46] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Additional data - ISO
	p.Fields[47] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Additional data - national
	p.Fields[48] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Additional data - private
	p.Fields[49] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 3}                  // Currency code, transaction
	p.Fields[50] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 3}                  // Currency code, settlement
	p.Fields[51] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 3}                  // Currency code, cardholder billing
	p.Fields[52] = FieldDefinition{Type: B, LengthType: FIXED, MaxLength: 16}                   // Personal identification number data
	p.Fields[53] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 16}                   // Security related control information
	p.Fields[54] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 120}               // Additional amounts
	p.Fields[55] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Reserved ISO
	p.Fields[56] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Reserved ISO
	p.Fields[57] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Reserved national
	p.Fields[58] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Reserved national
	p.Fields[59] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Reserved private
	p.Fields[60] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Reserved private
	p.Fields[61] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Reserved private
	p.Fields[62] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Reserved private
	p.Fields[63] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}               // Reserved private
	p.Fields[64] = FieldDefinition{Type: B, LengthType: FIXED, MaxLength: 16}                   // Message authentication code
	p.Fields[65] = FieldDefinition{Type: B, LengthType: FIXED, MaxLength: 16}                   // Bitmap, extended
	p.Fields[66] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 1}                    // Settlement code
	p.Fields[67] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 2}                    // Extended payment code
	p.Fields[68] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 3}                    // Receiving institution country code
	p.Fields[69] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 3}                    // Settlement institution country code
	p.Fields[70] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 3}                    // Network management information code
	p.Fields[71] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 4}                    // Message number
	p.Fields[72] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 4}                    // Message number, last
	p.Fields[73] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 6}                    // Date, action
	p.Fields[74] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 10}                   // Credits, number
	p.Fields[75] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 10}                   // Credits, reversal number
	p.Fields[76] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 10}                   // Debits, number
	p.Fields[77] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 10}                   // Debits, reversal number
	p.Fields[78] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 10}                   // Transfer number
	p.Fields[79] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 10}                   // Transfer, reversal number
	p.Fields[80] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 10}                   // Inquiries number
	p.Fields[81] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 10}                   // Authorizations, number
	p.Fields[82] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 12}                   // Credits, processing fee amount
	p.Fields[83] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 12}                   // Credits, transaction fee amount
	p.Fields[84] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 12}                   // Debits, processing fee amount
	p.Fields[85] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 12}                   // Debits, transaction fee amount
	p.Fields[86] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 16}                   // Credits, amount
	p.Fields[87] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 16}                   // Credits, reversal amount
	p.Fields[88] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 16}                   // Debits, amount
	p.Fields[89] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 16}                   // Debits, reversal amount
	p.Fields[90] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 42}                   // Original data elements
	p.Fields[91] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 1}                  // File update code
	p.Fields[92] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 2}                  // File security code
	p.Fields[93] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 5}                  // Response indicator
	p.Fields[94] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 7}                  // Service indicator
	p.Fields[95] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 42}                 // Replacement amounts
	p.Fields[96] = FieldDefinition{Type: B, LengthType: FIXED, MaxLength: 16}                   // Message security code
	p.Fields[97] = FieldDefinition{Type: N, LengthType: FIXED, MaxLength: 16}                   // Amount, net settlement
	p.Fields[98] = FieldDefinition{Type: ANS, LengthType: FIXED, MaxLength: 25}                 // Payee
	p.Fields[99] = FieldDefinition{Type: N, LengthType: LLVAR, MaxLength: 11}                   // Settlement institution identification code
	p.Fields[100] = FieldDefinition{Type: N, LengthType: LLVAR, MaxLength: 11}                  // Receiving institution identification code
	p.Fields[101] = FieldDefinition{Type: ANS, LengthType: LLVAR, MaxLength: 17}                // File name
	p.Fields[102] = FieldDefinition{Type: ANS, LengthType: LLVAR, MaxLength: 28}                // Account identification 1
	p.Fields[103] = FieldDefinition{Type: ANS, LengthType: LLVAR, MaxLength: 28}                // Account identification 2
	p.Fields[104] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 100}              // Transaction description
	p.Fields[105] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for ISO use
	p.Fields[106] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for ISO use
	p.Fields[107] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for ISO use
	p.Fields[108] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for ISO use
	p.Fields[109] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for ISO use
	p.Fields[110] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for ISO use
	p.Fields[111] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for ISO use
	p.Fields[112] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for national use
	p.Fields[113] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for national use
	p.Fields[114] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for national use
	p.Fields[115] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for national use
	p.Fields[116] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for national use
	p.Fields[117] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for national use
	p.Fields[118] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for national use
	p.Fields[119] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for national use
	p.Fields[120] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for private use
	p.Fields[121] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for private use
	p.Fields[122] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for private use
	p.Fields[123] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for private use
	p.Fields[124] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for private use
	p.Fields[125] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for private use
	p.Fields[126] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for private use
	p.Fields[127] = FieldDefinition{Type: ANS, LengthType: LLLVAR, MaxLength: 999}              // Reserved for private use
	p.Fields[128] = FieldDefinition{Type: B, LengthType: FIXED, MaxLength: 16}                  // Message authentication code
	return p
}

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

// Message represents an ISO8583 message
type Message struct {
	MTI         []byte
	Bitmap      [129]bool   // Index 0 unused, 1-128 for fields
	Fields      [129][]byte // Index 0 unused, 1-128 for fields
	Packager    *Packager
	FullMessage []byte
}

// NewMessage creates a new ISO8583 message with a given packager.
func NewMessage(packager *Packager) *Message {
	if packager == nil {
		panic("packager cannot be nil")
	}
	return &Message{
		Packager: packager,
	}
}

// Validate checks if all mandatory fields are present in the message.
func (m *Message) Validate() error {
	for i := 1; i <= 128; i++ {
		fieldDef := m.Packager.Fields[i]
		if fieldDef.IsMandatory && !m.Bitmap[i] {
			return fmt.Errorf("%w: field %d", ErrMissingMandatoryField, i)
		}
	}
	return nil
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

	// Parse primary bitmap (Field 1)
	bitmapFieldDef := m.Packager.Fields[1]
	if bitmapFieldDef.Type != B || bitmapFieldDef.LengthType != FIXED {
		return fmt.Errorf("field 1 (bitmap) must be of type 'b' and 'FIXED' length")
	}
	bitmapHexLen := bitmapFieldDef.MaxLength
	if len(data) < pos+bitmapHexLen {
		return ErrInvalidBitmap
	}

	var bitmap1 [8]byte
	_, err := hex.Decode(bitmap1[:], data[pos:pos+bitmapHexLen])
	if err != nil {
		return fmt.Errorf("invalid primary bitmap hex: %w", err)
	}
	pos += bitmapHexLen

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
		if len(data) < pos+bitmapHexLen {
			return ErrInvalidBitmap
		}
		var bitmap2 [8]byte
		_, err := hex.Decode(bitmap2[:], data[pos:pos+bitmapHexLen])
		if err != nil {
			return fmt.Errorf("invalid secondary bitmap hex: %w", err)
		}
		pos += bitmapHexLen

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

		fieldDef := m.Packager.Fields[fieldNum]
		if fieldDef.MaxLength == 0 { // Field not defined
			continue
		}

		switch fieldDef.LengthType {
		case FIXED:
			fieldLen := fieldDef.MaxLength
			if len(data) < pos+fieldLen {
				return fmt.Errorf("insufficient data for field %d", fieldNum)
			}
			m.Fields[fieldNum] = data[pos : pos+fieldLen] // Zero-copy
			pos += fieldLen
		case LLVAR, LLLVAR:
			lenDigits := 2
			if fieldDef.LengthType == LLLVAR {
				lenDigits = 3
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
			m.Fields[fieldNum] = data[pos : pos+fieldLen] // Zero-copy
			pos += fieldLen
		default:
			return fmt.Errorf("unsupported length type for field %d: %s", fieldNum, fieldDef.LengthType)
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

	// Pack bitmap (Field 1)
	bitmapFieldDef := m.Packager.Fields[1]
	if bitmapFieldDef.Type != B || bitmapFieldDef.LengthType != FIXED {
		return 0, fmt.Errorf("field 1 (bitmap) must be of type 'b' and 'FIXED' length")
	}
	bitmapHexLen := bitmapFieldDef.MaxLength
	if len(buf) < pos+bitmapHexLen {
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
	hex.Encode(buf[pos:pos+bitmapHexLen], bitmap1[:])
	pos += bitmapHexLen

	if needSecondary {
		if len(buf) < pos+bitmapHexLen {
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
		hex.Encode(buf[pos:pos+bitmapHexLen], bitmap2[:])
		pos += bitmapHexLen
	}

	// Pack fields
	for fieldNum := 2; fieldNum <= 128; fieldNum++ {
		if !m.Bitmap[fieldNum] {
			continue
		}
		fieldDef := m.Packager.Fields[fieldNum]
		if fieldDef.MaxLength == 0 {
			continue
		}
		fieldData := m.Fields[fieldNum]

		switch fieldDef.LengthType {
		case FIXED:
			fieldLen := fieldDef.MaxLength
			if len(buf) < pos+fieldLen {
				return 0, ErrBufferTooSmall
			}
			copy(buf[pos:pos+fieldLen], fieldData)
			// Apply padding
			if len(fieldData) < fieldLen {
				padding := buf[pos+len(fieldData) : pos+fieldLen]
				padChar := byte(' ')
				if fieldDef.Type == N {
					padChar = byte('0')
				}
				for i := range padding {
					padding[i] = padChar
				}
			}
			pos += fieldLen

		case LLVAR, LLLVAR:
			lenDigits := 2
			if fieldDef.LengthType == LLLVAR {
				lenDigits = 3
			}
			fieldLen := len(fieldData)
			if len(buf) < pos+lenDigits+fieldLen {
				return 0, ErrBufferTooSmall
			}
			writeIntToASCII(buf[pos:pos+lenDigits], fieldLen, lenDigits)
			pos += lenDigits
			copy(buf[pos:pos+fieldLen], fieldData)
			pos += fieldLen
		default:
			return 0, fmt.Errorf("unsupported length type for field %d: %s", fieldNum, fieldDef.LengthType)
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
	m.Fields[fieldNum] = make([]byte, len(data))
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
