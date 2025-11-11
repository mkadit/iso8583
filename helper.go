package iso8583

// Add this lookup table at the top of bitmap.go
const hexTableUpper = "0123456789ABCDEF"

// encodeHexUpper converts src to uppercase hex and writes it to dst.
func encodeHexUpper(dst, src []byte) {
	for i, v := range src {
		dst[i*2] = hexTableUpper[v>>4]
		dst[i*2+1] = hexTableUpper[v&0x0f]
	}
}

// Simple power function, since math.Pow is float64
func pow(a, b float64) float64 {
	// This is just a quick substitute for math.Pow
	res := 1.0
	for i := 0; i < int(b); i++ {
		res *= a
	}
	return res
}
