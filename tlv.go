package iso8583

import (
	"fmt"
	"strconv"
	"sync"
)

// TLVParser handles parsing and packing of Tag-Length-Value encoded data.
// It supports Standard (1-byte T/L), EMV (variable T/L), and a custom
// fixed-length ASCII format.
// A TLVParser instance is stateful (it contains a buffer) but is safe
// for concurrent use due to its internal mutex.
type TLVParser struct {
	tlvType TLVType
	buffer  []TLV // Internal buffer to reduce allocations during parsing
	mu      sync.Mutex

	asciiTagLen     int // e.g., 2 for "AL"
	asciiLenLen     int // e.g., 2 for "04"
	asciiLengthBase int // 10 for decimal, 16 for hex
}

// NewTLVParser creates a new TLV parser for Standard or EMV types.
func NewTLVParser(tlvType TLVType) *TLVParser {
	return &TLVParser{
		tlvType: tlvType,
		buffer:  make([]TLV, 0, 32), // Pre-allocate for common case
	}
}

// NewASCIITLVParser creates a new parser for fixed-length ASCII TLV.
// tagLen: number of characters for the tag (e.g., 2 for "AL")
// lenLen: number of characters for the length (e.g., 2 for "04")
// base:   10 for decimal length ("04"), 16 for hex length ("0C")
func NewASCIITLVParser(tagLen, lenLen, base int) *TLVParser {
	return &TLVParser{
		tlvType:         TLVASCII,
		buffer:          make([]TLV, 0, 32),
		asciiTagLen:     tagLen,
		asciiLenLen:     lenLen,
		asciiLengthBase: base,
	}
}

// reset clears the parser's internal buffer for reuse.
func (tp *TLVParser) reset() {
	tp.buffer = tp.buffer[:0] // Reset buffer slice length

	// Reset new fields
	tp.asciiTagLen = 0
	tp.asciiLenLen = 0
	tp.asciiLengthBase = 0
}

// ParseTLV parses TLV data from a byte slice based on the parser's configured type.
func (tp *TLVParser) ParseTLV(data []byte) ([]TLV, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tp.buffer = tp.buffer[:0] // Reset buffer for this parse operation

	switch tp.tlvType {
	case TLVStandard:
		return tp.parseStandardTLV(data)
	case TLVEMV:
		return tp.parseEMVTLV(data)
	case TLVASCII:
		return tp.parseASCIITLV(data)
	default:
		return nil, fmt.Errorf("unsupported TLV type")
	}
}

// parseASCIITLV parses fixed-length ASCII TLV format.
// Format: T(fixed_ascii_len) L(fixed_ascii_len) V(variable_len)
// Example: "AL04Data" (Tag="AL", Length="04", Value="Data")
func (tp *TLVParser) parseASCIITLV(data []byte) ([]TLV, error) {
	if tp.asciiTagLen <= 0 || tp.asciiLenLen <= 0 {
		return nil, fmt.Errorf("ASCII TLV parser not configured (tag/length len is zero)")
	}

	tagLen := tp.asciiTagLen
	lenLen := tp.asciiLenLen

	offset := 0
	for offset < len(data) {
		// Parse Tag (fixed ASCII length)
		if offset+tagLen > len(data) {
			if offset == len(data) {
				break // Cleanly finished at the end
			}
			return nil, fmt.Errorf("insufficient data for tag at offset %d: need %d, got %d", offset, tagLen, len(data)-offset)
		}
		tag := data[offset : offset+tagLen]
		offset += tagLen

		// Parse Length (fixed ASCII length)
		if offset+lenLen > len(data) {
			return nil, fmt.Errorf("insufficient data for length at offset %d: need %d, got %d", offset, lenLen, len(data)-offset)
		}

		// Use strconv.ParseInt for base 10/16
		lengthStr := string(data[offset : offset+lenLen])
		length, err := strconv.ParseInt(lengthStr, tp.asciiLengthBase, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid ASCII length '%s': %w", lengthStr, err)
		}
		offset += lenLen

		// Parse Value
		if offset+int(length) > len(data) {
			return nil, fmt.Errorf("insufficient data for value at offset %d: need %d, got %d", offset, length, len(data)-offset)
		}
		value := data[offset : offset+int(length)]
		offset += int(length)

		tlv := TLV{
			Tag:    tag,
			Length: int(length),
			Value:  value,
		}

		tp.buffer = append(tp.buffer, tlv)
	}

	// Return a copy of the buffer's contents, not the buffer itself
	result := make([]TLV, len(tp.buffer))
	copy(result, tp.buffer)
	return result, nil
}

// parseStandardTLV parses standard TLV format (T=1byte, L=1byte, V=variable).
func (tp *TLVParser) parseStandardTLV(data []byte) ([]TLV, error) {
	offset := 0
	for offset < len(data) {
		// Parse Tag (1 byte)
		if offset >= len(data) {
			break // Reached end
		}
		tag := data[offset : offset+1]
		offset++

		// Parse Length (1 byte)
		if offset >= len(data) {
			return nil, ErrInvalidTLV // Truncated message
		}
		length := int(data[offset])
		offset++

		// Parse Value
		if offset+length > len(data) {
			return nil, ErrInvalidTLV // Truncated message
		}
		value := data[offset : offset+length]
		offset += length

		tlv := TLV{
			Tag:    tag,
			Length: length,
			Value:  value,
		}

		tp.buffer = append(tp.buffer, tlv)
	}

	// Return copy of buffer
	result := make([]TLV, len(tp.buffer))
	copy(result, tp.buffer)
	return result, nil
}

// parseEMVTLV parses EMV TLV format with variable tag and length encoding.
func (tp *TLVParser) parseEMVTLV(data []byte) ([]TLV, error) {
	offset := 0
	for offset < len(data) {
		// Parse Tag (variable length)
		tagStart := offset
		if offset >= len(data) {
			break
		}

		// First byte of tag
		firstByte := data[offset]
		offset++

		// Check if tag continues: bits 5-1 of first byte are 11111
		if (firstByte & 0x1F) == 0x1F {
			// --- FIX IS HERE ---
			// Keep reading as long as the MSB of the *current* byte is 1
			// This means more tag bytes follow.
			for offset < len(data) && (data[offset]&0x80) != 0 {
				offset++ // Consume this byte (e.g., 9F -> 80)
			}
			// Now we are at the last byte (MSB is 0). Consume it.
			if offset >= len(data) {
				return nil, ErrInvalidTLV // Truncated tag
			}
			offset++ // Consume the final byte of the tag (e.g., 33 in 9F33)
			// --- END FIX ---
		}

		if offset > len(data) {
			return nil, ErrInvalidTLV
		}
		tag := data[tagStart:offset]

		// Parse Length (variable length)
		if offset >= len(data) {
			return nil, ErrInvalidTLV
		}

		lengthByte := data[offset]
		offset++
		var length int

		if (lengthByte & 0x80) == 0 {
			// Short form - length is in this single byte (0-127)
			length = int(lengthByte)
		} else {
			// Long form - first byte's lower 7 bits indicate
			// number of subsequent length bytes.
			numLengthBytes := int(lengthByte & 0x7F)
			if numLengthBytes == 0 || numLengthBytes > 4 {
				return nil, ErrInvalidTLV // Invalid number of length bytes
			}

			if offset+numLengthBytes > len(data) {
				return nil, ErrInvalidTLV // Truncated length
			}

			// Read the N-byte length
			length = 0
			for i := 0; i < numLengthBytes; i++ {
				length = (length << 8) | int(data[offset])
				offset++
			}
		}

		// Parse Value
		if offset+length > len(data) {
			return nil, ErrInvalidTLV // Truncated value
		}
		value := data[offset : offset+length]
		offset += length

		tlv := TLV{
			Tag:    tag,
			Length: length,
			Value:  value,
		}

		tp.buffer = append(tp.buffer, tlv)
	}

	// Return copy of buffer
	result := make([]TLV, len(tp.buffer))
	copy(result, tp.buffer)
	return result, nil
}

// PackTLV packs a slice of TLV structs into a byte buffer.
func (tp *TLVParser) PackTLV(tlvs []TLV, buf []byte) (int, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	switch tp.tlvType {
	case TLVStandard:
		return tp.packStandardTLV(tlvs, buf)
	case TLVEMV:
		return tp.packEMVTLV(tlvs, buf)
	case TLVASCII:
		return tp.packASCIITLV(tlvs, buf)
	default:
		return 0, fmt.Errorf("unsupported TLV type")
	}
}

// packASCIITLV packs fixed-length ASCII TLV format.
func (tp *TLVParser) packASCIITLV(tlvs []TLV, buf []byte) (int, error) {
	if tp.asciiTagLen <= 0 || tp.asciiLenLen <= 0 {
		return 0, fmt.Errorf("ASCII TLV parser not configured (tag/length len is zero)")
	}

	tagLen := tp.asciiTagLen
	lenLen := tp.asciiLenLen
	offset := 0

	for _, tlv := range tlvs {
		// Check buffer space
		requiredSpace := tagLen + lenLen + len(tlv.Value)
		if offset+requiredSpace > len(buf) {
			return 0, ErrBufferTooSmall
		}

		// Pack Tag
		if len(tlv.Tag) != tagLen {
			return 0, fmt.Errorf("ASCII TLV tag length mismatch: expected %d, got %d for tag %s", tagLen, len(tlv.Tag), tlv.Tag)
		}
		copy(buf[offset:], tlv.Tag)
		offset += tagLen

		// Pack Length
		valueLen := len(tlv.Value)

		// Format string for length (e.g., "%02d" for lenLen=2, base=10)
		var format string
		if tp.asciiLengthBase == 16 {
			format = fmt.Sprintf("%%0%dX", lenLen) // e.g., %02X
		} else {
			format = fmt.Sprintf("%%0%dd", lenLen) // e.g., %02d
		}

		// Check max length
		// Note: This pow(float64) is not ideal for performance.
		// A pre-calculated map or switch would be faster.
		maxLen := int(pow(float64(tp.asciiLengthBase), float64(lenLen))) - 1
		if valueLen > maxLen {
			return 0, fmt.Errorf("ASCII TLV value length %d exceeds maximum %d for %d digits", valueLen, maxLen, lenLen)
		}

		// Use Sprintf to format the length string (e.g., 4 -> "04")
		lengthStr := fmt.Sprintf(format, valueLen)
		copy(buf[offset:], lengthStr)
		offset += lenLen

		// Pack Value
		copy(buf[offset:], tlv.Value)
		offset += len(tlv.Value)
	}

	return offset, nil
}

// packStandardTLV packs standard TLV format (T=1byte, L=1byte, V=variable).
func (tp *TLVParser) packStandardTLV(tlvs []TLV, buf []byte) (int, error) {
	offset := 0

	for _, tlv := range tlvs {
		// Check buffer space
		requiredSpace := len(tlv.Tag) + 1 + len(tlv.Value) // Tag + 1-byte L + Value
		if offset+requiredSpace > len(buf) {
			return 0, ErrBufferTooSmall
		}

		// Pack Tag (must be 1 byte)
		if len(tlv.Tag) != 1 {
			return 0, fmt.Errorf("standard TLV tag must be 1 byte")
		}
		buf[offset] = tlv.Tag[0]
		offset++

		// Pack Length (1 byte)
		if len(tlv.Value) > 255 {
			return 0, fmt.Errorf("standard TLV value too long (max 255)")
		}
		buf[offset] = byte(len(tlv.Value))
		offset++

		// Pack Value
		copy(buf[offset:], tlv.Value)
		offset += len(tlv.Value)
	}

	return offset, nil
}

// packEMVTLV packs EMV TLV format (variable T/L).
func (tp *TLVParser) packEMVTLV(tlvs []TLV, buf []byte) (int, error) {
	offset := 0

	for _, tlv := range tlvs {
		// Pack Tag
		if offset+len(tlv.Tag) > len(buf) {
			return 0, ErrBufferTooSmall
		}
		copy(buf[offset:], tlv.Tag)
		offset += len(tlv.Tag)

		// Pack Length
		valueLen := len(tlv.Value)
		if valueLen < 0x80 {
			// Short form (0-127)
			if offset+1 > len(buf) {
				return 0, ErrBufferTooSmall
			}
			buf[offset] = byte(valueLen)
			offset++
		} else {
			// Long form
			var lengthBytes []byte
			temp := valueLen
			// Build length bytes in reverse (big-endian)
			for temp > 0 {
				lengthBytes = append([]byte{byte(temp & 0xFF)}, lengthBytes...)
				temp >>= 8
			}

			// Check space for 0x8N byte + N length bytes
			if offset+1+len(lengthBytes) > len(buf) {
				return 0, ErrBufferTooSmall
			}

			// Write 0x8N byte (e.g., 0x81 for 1 byte, 0x82 for 2 bytes)
			buf[offset] = byte(0x80 | len(lengthBytes))
			offset++
			// Write the actual length bytes
			copy(buf[offset:], lengthBytes)
			offset += len(lengthBytes)
		}

		// Pack Value
		if offset+len(tlv.Value) > len(buf) {
			return 0, ErrBufferTooSmall
		}
		copy(buf[offset:], tlv.Value)
		offset += len(tlv.Value)
	}

	return offset, nil
}

// FindTLV finds the first TLV entry matching the given tag.
func FindTLV(tlvs []TLV, tag []byte) (*TLV, bool) {
	for i := range tlvs {
		if len(tlvs[i].Tag) == len(tag) {
			match := true
			// Simple byte-by-byte comparison
			for j := range tag {
				if tlvs[i].Tag[j] != tag[j] {
					match = false
					break
				}
			}
			if match {
				return &tlvs[i], true
			}
		}
	}
	return nil, false
}

// FilterTLVsByTag finds all TLV entries matching the given tag prefix.
func FilterTLVsByTag(tlvs []TLV, tagPrefix []byte) []TLV {
	var result []TLV

	for _, tlv := range tlvs {
		if len(tlv.Tag) >= len(tagPrefix) {
			match := true
			// Check prefix
			for i := range tagPrefix {
				if tlv.Tag[i] != tagPrefix[i] {
					match = false
					break
				}
			}
			if match {
				result = append(result, tlv)
			}
		}
	}

	return result
}

// --- MODIFIED ---
// TLVToMap converts a slice of TLV structs to a map[string][]byte.
// For ASCII TLV, the map key is the literal tag string (e.g., "AL").
// For Standard/EMV TLV, the map key is a HEX string (e.g., "9F02").
func TLVToMap(tlvs []TLV, tlvType TLVType) map[string][]byte {
	result := make(map[string][]byte)

	for _, tlv := range tlvs {
		// --- MODIFIED ---
		var key string
		if tlvType == TLVASCII {
			key = string(tlv.Tag) // Key is literal string
		} else {
			key = fmt.Sprintf("%X", tlv.Tag) // Key is hex string
		}
		// --- END MODIFIED ---
		result[key] = tlv.Value
	}

	return result
}

// --- MODIFIED ---
// MapToTLV converts a map[string][]byte to a slice of TLV structs.
// For ASCII TLV, the map key is the literal tag string (e.g., "AL").
// For Standard/EMV TLV, the map key is a HEX string (e.g., "9F02").
func MapToTLV(tlvMap map[string][]byte, tlvType TLVType) ([]TLV, error) {
	var result []TLV

	for tagStr, value := range tlvMap {
		// --- MODIFIED ---
		var tag []byte

		if tlvType == TLVASCII {
			// For ASCII, the key is the tag itself
			tag = []byte(tagStr)
		} else {
			// For Standard/EMV, the key is a hex string, needs decoding
			tag = make([]byte, len(tagStr)/2)
			for i := 0; i < len(tagStr); i += 2 {
				var b byte
				// Manual hex decoding
				if tagStr[i] >= '0' && tagStr[i] <= '9' {
					b = (tagStr[i] - '0') << 4
				} else if tagStr[i] >= 'A' && tagStr[i] <= 'F' {
					b = (tagStr[i] - 'A' + 10) << 4
				} else if tagStr[i] >= 'a' && tagStr[i] <= 'f' {
					b = (tagStr[i] - 'a' + 10) << 4
				} else {
					return nil, fmt.Errorf("invalid hex character in tag")
				}

				if i+1 < len(tagStr) {
					if tagStr[i+1] >= '0' && tagStr[i+1] <= '9' {
						b |= tagStr[i+1] - '0'
					} else if tagStr[i+1] >= 'A' && tagStr[i+1] <= 'F' {
						b |= tagStr[i+1] - 'A' + 10
					} else if tagStr[i+1] >= 'a' && tagStr[i+1] <= 'f' {
						b |= tagStr[i+1] - 'a' + 10
					} else {
						return nil, fmt.Errorf("invalid hex character in tag")
					}
				}
				tag[i/2] = b
			}
		}
		// --- END MODIFIED ---

		result = append(result, TLV{
			Tag:    tag,
			Length: len(value),
			Value:  value,
		})
	}

	return result, nil
}

// TLVToMapString is a convenience function that converts a TLV slice to a
// map[string]string, converting all values to strings.
// For ASCII TLV, the map key is the literal tag string (e.g., "AL").
// For Standard/EMV TLV, the map key is a HEX string (e.g., "9F02").
func TLVToMapString(tlvs []TLV, tlvType TLVType) map[string]string {
	result := make(map[string]string)

	for _, tlv := range tlvs {
		var key string
		if tlvType == TLVASCII {
			key = string(tlv.Tag)
		} else {
			key = fmt.Sprintf("%X", tlv.Tag)
		}
		// Convert the byte slice value to a string
		result[key] = string(tlv.Value)
	}

	return result
}
