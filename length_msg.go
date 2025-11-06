package iso8583

import (
	"fmt"
	"strconv"
)

// WriteLengthIndicator writes the message length indicator (the prefix that
// tells a TCP server how long the message is) to the buffer.
// Returns the number of bytes written.
func WriteLengthIndicator(msgLen int, buf []byte, config LengthIndicatorConfig) (int, error) {
	if config.Type == LengthIndicatorNone {
		return 0, nil
	}

	if len(buf) < config.Length {
		return 0, ErrBufferTooSmall
	}

	switch config.Type {
	case LengthIndicatorBinary:
		return writeBinaryLengthIndicator(msgLen, buf, config)
	case LengthIndicatorASCII:
		return writeASCIILengthIndicator(msgLen, buf, config)
	case LengthIndicatorHex:
		return writeHexLengthIndicator(msgLen, buf, config)
	default:
		return 0, fmt.Errorf("unsupported length indicator type")
	}
}

// ReadLengthIndicator reads the message length indicator from the buffer.
// Returns:
// 1. The message length (e.g., 200 for "0200")
// 2. The number of bytes consumed by the indicator (e.g., 4 for "0200")
// 3. An error, if any
func ReadLengthIndicator(buf []byte, config LengthIndicatorConfig) (int, int, error) {
	if config.Type == LengthIndicatorNone {
		// No length indicator, assume the buffer is the full message
		return len(buf), 0, nil
	}

	if len(buf) < config.Length {
		return 0, 0, ErrInvalidLength
	}

	switch config.Type {
	case LengthIndicatorBinary:
		return readBinaryLengthIndicator(buf, config)
	case LengthIndicatorASCII:
		return readASCIILengthIndicator(buf, config)
	case LengthIndicatorHex:
		return readHexLengthIndicator(buf, config)
	default:
		return 0, 0, fmt.Errorf("unsupported length indicator type")
	}
}

// writeBinaryLengthIndicator writes binary length (2 or 4 bytes, big-endian).
func writeBinaryLengthIndicator(msgLen int, buf []byte, config LengthIndicatorConfig) (int, error) {
	switch config.Length {
	case 2:
		// 2-byte binary (max 0xFFFF)
		if msgLen > 0xFFFF {
			return 0, fmt.Errorf("message length %d exceeds 2-byte maximum", msgLen)
		}
		buf[0] = byte(msgLen >> 8) // High byte
		buf[1] = byte(msgLen)      // Low byte
		return 2, nil

	case 4:
		// 4-byte binary (max 0x7FFFFFFF)
		if msgLen > 0x7FFFFFFF {
			return 0, fmt.Errorf("message length %d exceeds 4-byte maximum", msgLen)
		}
		buf[0] = byte(msgLen >> 24)
		buf[1] = byte(msgLen >> 16)
		buf[2] = byte(msgLen >> 8)
		buf[3] = byte(msgLen)
		return 4, nil

	default:
		return 0, fmt.Errorf("invalid binary length indicator size: %d (must be 2 or 4)", config.Length)
	}
}

// readBinaryLengthIndicator reads binary length (2 or 4 bytes, big-endian).
func readBinaryLengthIndicator(buf []byte, config LengthIndicatorConfig) (int, int, error) {
	switch config.Length {
	case 2:
		if len(buf) < 2 {
			return 0, 0, ErrInvalidLength
		}
		msgLen := int(buf[0])<<8 | int(buf[1]) // Combine high and low bytes
		return msgLen, 2, nil

	case 4:
		if len(buf) < 4 {
			return 0, 0, ErrInvalidLength
		}
		msgLen := int(buf[0])<<24 | int(buf[1])<<16 | int(buf[2])<<8 | int(buf[3])
		return msgLen, 4, nil

	default:
		return 0, 0, fmt.Errorf("invalid binary length indicator size: %d (must be 2 or 4)", config.Length)
	}
}

// writeASCIILengthIndicator writes ASCII decimal length (typically 4 digits, e.g., "0200").
func writeASCIILengthIndicator(msgLen int, buf []byte, config LengthIndicatorConfig) (int, error) {
	// This implementation assumes a 4-char ASCII length, which is common.
	if config.Length != 4 {
		return 0, fmt.Errorf("ASCII length indicator must be 4 characters, got %d", config.Length)
	}

	if msgLen > 9999 {
		return 0, fmt.Errorf("message length %d exceeds 4-digit ASCII maximum", msgLen)
	}

	// Write as zero-padded decimal (e.g., 200 -> "0200")
	// Note: writeIntToASCII is defined in message.go
	writeIntToASCII(buf[:4], msgLen, 4)
	return 4, nil
}

// readASCIILengthIndicator reads ASCII decimal length (typically 4 digits, e.g., "0200").
func readASCIILengthIndicator(buf []byte, config LengthIndicatorConfig) (int, int, error) {
	if config.Length != 4 {
		return 0, 0, fmt.Errorf("ASCII length indicator must be 4 characters, got %d", config.Length)
	}

	if len(buf) < 4 {
		return 0, 0, ErrInvalidLength
	}

	// Use a fast, allocation-free ASCII parser
	msgLen, err := parseASCIIToInt(buf[:4])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid ASCII length indicator: %w", err)
	}

	return msgLen, 4, nil
}

// writeHexLengthIndicator writes hexadecimal ASCII length (typically 4 chars, e.g., "00C8" for 200).
func writeHexLengthIndicator(msgLen int, buf []byte, config LengthIndicatorConfig) (int, error) {
	if config.Length != 4 {
		return 0, fmt.Errorf("hex length indicator must be 4 characters, got %d", config.Length)
	}

	if msgLen > 0xFFFF {
		return 0, fmt.Errorf("message length %d exceeds 4-char hex maximum", msgLen)
	}

	// Convert to hex string (e.g., 200 -> "00C8")
	hexStr := fmt.Sprintf("%04X", msgLen)
	copy(buf[:4], hexStr)
	return 4, nil
}

// readHexLengthIndicator reads hexadecimal ASCII length (typically 4 chars, e.g., "00C8").
func readHexLengthIndicator(buf []byte, config LengthIndicatorConfig) (int, int, error) {
	if config.Length != 4 {
		return 0, 0, fmt.Errorf("hex length indicator must be 4 characters, got %d", config.Length)
	}

	if len(buf) < 4 {
		return 0, 0, ErrInvalidLength
	}

	// Parse hex string (e.g., "00C8" -> 200)
	msgLen, err := strconv.ParseInt(string(buf[:4]), 16, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hex length indicator: %w", err)
	}

	return int(msgLen), 4, nil
}

// parseASCIIToInt is a helper function to parse ASCII digits to an integer
// without using strconv, avoiding allocations.
func parseASCIIToInt(b []byte) (int, error) {
	n := 0
	for _, ch := range b {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid character '%c' in numeric string", ch)
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}
