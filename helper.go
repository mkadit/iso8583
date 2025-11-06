package iso8583

// Simple power function, since math.Pow is float64
func pow(a, b float64) float64 {
	// This is just a quick substitute for math.Pow
	res := 1.0
	for i := 0; i < int(b); i++ {
		res *= a
	}
	return res
}
