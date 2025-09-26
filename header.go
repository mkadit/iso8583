package iso8583

// HeaderType defines how ISO8583 message length headers are encoded.
type HeaderType int

const (
	HeaderNone   HeaderType = iota
	HeaderBinary            // 2-byte binary length
	HeaderASCII             // 4-digit ASCII decimal length, e.g. "0048"
	HeaderHex               // 4-char ASCII hex length, e.g. "0030"
)

// WriteHeader writes the message length into buf according to the header type.
// Returns the number of bytes written or 0 on error.
func WriteHeader(msgLen int, buf []byte, htype HeaderType) (int, error) {
	switch htype {
	case HeaderNone:
		return 0, nil

	case HeaderBinary:
		if len(buf) < 2 || msgLen > 0xFFFF {
			return 0, ErrInvalidLength
		}
		buf[0] = byte(msgLen >> 8)
		buf[1] = byte(msgLen & 0xFF)
		return 2, nil

	case HeaderASCII:
		if len(buf) < 4 || msgLen > 9999 {
			return 0, ErrInvalidLength
		}
		buf[0] = byte('0' + (msgLen/1000)%10)
		buf[1] = byte('0' + (msgLen/100)%10)
		buf[2] = byte('0' + (msgLen/10)%10)
		buf[3] = byte('0' + msgLen%10)
		return 4, nil

	case HeaderHex:
		if len(buf) < 4 || msgLen > 0xFFFF {
			return 0, ErrInvalidLength
		}
		hi := (msgLen >> 8) & 0xFF
		lo := msgLen & 0xFF
		const hexChars = "0123456789ABCDEF"
		buf[0] = hexChars[hi>>4]
		buf[1] = hexChars[hi&0xF]
		buf[2] = hexChars[lo>>4]
		buf[3] = hexChars[lo&0xF]
		return 4, nil
	}
	return 0, ErrInvalidLength
}

// ReadHeader reads the message length from buf according to the header type.
func ReadHeader(buf []byte, htype HeaderType) (int, error) {
	switch htype {
	case HeaderNone:
		return 0, nil

	case HeaderBinary:
		if len(buf) < 2 {
			return 0, ErrInsufficientData
		}
		return int(buf[0])<<8 | int(buf[1]), nil

	case HeaderASCII:
		if len(buf) < 4 {
			return 0, ErrInsufficientData
		}
		n := 0
		for i := 0; i < 4; i++ {
			if buf[i] < '0' || buf[i] > '9' {
				return 0, ErrInvalidLength
			}
			n = n*10 + int(buf[i]-'0')
		}
		return n, nil

	case HeaderHex:
		if len(buf) < 4 {
			return 0, ErrInsufficientData
		}
		hi, err := fromHex(buf[0], buf[1])
		if err != nil {
			return 0, err
		}
		lo, err := fromHex(buf[2], buf[3])
		if err != nil {
			return 0, err
		}
		return int(hi)<<8 | int(lo), nil
	}
	return 0, ErrInvalidLength
}

func fromHex(b1, b2 byte) (byte, error) {
	hexVal := func(b byte) (int, bool) {
		switch {
		case b >= '0' && b <= '9':
			return int(b - '0'), true
		case b >= 'A' && b <= 'F':
			return int(b - 'A' + 10), true
		case b >= 'a' && b <= 'f':
			return int(b - 'a' + 10), true
		}
		return 0, false
	}
	hi, ok1 := hexVal(b1)
	lo, ok2 := hexVal(b2)
	if !ok1 || !ok2 {
		return 0, ErrInvalidLength
	}
	return byte(hi<<4 | lo), nil
}
