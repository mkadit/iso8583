package iso8583

// TLV represents a Tag-Length-Value element (zero-copy slices).
type TLV struct {
	Tag   []byte
	Value []byte
}

// ParseTLV parses a buffer of TLV-encoded data into zero-copy slices.
// Supports EMV BER-TLV encoding (1–3 byte tags, BER length).
func ParseTLV(buf []byte) ([]TLV, error) {
	var result []TLV
	pos := 0
	for pos < len(buf) {
		// Parse Tag (1–3 bytes)
		tagStart := pos
		pos++
		if buf[tagStart]&0x1F == 0x1F {
			if pos >= len(buf) {
				return nil, ErrInsufficientData
			}
			pos++
			if buf[pos-1]&0x80 != 0 {
				if pos >= len(buf) {
					return nil, ErrInsufficientData
				}
				pos++
			}
		}
		tag := buf[tagStart:pos]

		// Parse Length (BER rules)
		if pos >= len(buf) {
			return nil, ErrInsufficientData
		}
		length := int(buf[pos])
		pos++
		if length&0x80 != 0 {
			numBytes := length & 0x7F
			if pos+numBytes > len(buf) {
				return nil, ErrInsufficientData
			}
			length = 0
			for i := 0; i < int(numBytes); i++ {
				length = (length << 8) | int(buf[pos])
				pos++
			}
		}

		// Parse Value
		if pos+length > len(buf) {
			return nil, ErrInsufficientData
		}
		value := buf[pos : pos+length]
		pos += length

		result = append(result, TLV{Tag: tag, Value: value})
	}
	return result, nil
}

// PackTLV serializes a TLV slice back into buf.
// Returns total bytes written.
func PackTLV(elements []TLV, buf []byte) (int, error) {
	pos := 0
	for _, e := range elements {
		// Tag
		if len(buf) < pos+len(e.Tag) {
			return 0, ErrBufferTooSmall
		}
		copy(buf[pos:], e.Tag)
		pos += len(e.Tag)

		// Length
		length := len(e.Value)
		if length < 0x80 {
			if pos >= len(buf) {
				return 0, ErrBufferTooSmall
			}
			buf[pos] = byte(length)
			pos++
		} else if length <= 0xFF {
			if len(buf) < pos+2 {
				return 0, ErrBufferTooSmall
			}
			buf[pos] = 0x81
			buf[pos+1] = byte(length)
			pos += 2
		} else if length <= 0xFFFF {
			if len(buf) < pos+3 {
				return 0, ErrBufferTooSmall
			}
			buf[pos] = 0x82
			buf[pos+1] = byte(length >> 8)
			buf[pos+2] = byte(length)
			pos += 3
		} else {
			return 0, ErrInvalidLength
		}

		// Value
		if len(buf) < pos+len(e.Value) {
			return 0, ErrBufferTooSmall
		}
		copy(buf[pos:], e.Value)
		pos += len(e.Value)
	}
	return pos, nil
}
