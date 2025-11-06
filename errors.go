package iso8583

import "fmt"

var (
	ErrInvalidMTI       = fmt.Errorf("invalid MTI")
	ErrInvalidField     = fmt.Errorf("invalid field")
	ErrFieldNotFound    = fmt.Errorf("field not found")
	ErrInvalidLength    = fmt.Errorf("invalid field length")
	ErrInvalidBitmap    = fmt.Errorf("invalid bitmap")
	ErrInvalidTLV       = fmt.Errorf("invalid TLV data")
	ErrValidationFailed = fmt.Errorf("validation failed")
	ErrBufferTooSmall   = fmt.Errorf("buffer too small")
	ErrInvalidHeader    = fmt.Errorf("invalid header")

	ErrNoPackagerConfigured  = fmt.Errorf("no packager configured")
	ErrFieldNotConfigured    = fmt.Errorf("field not configured")
	ErrUnsupportedLengthType = fmt.Errorf("unsupported length type")
	ErrInvalidBitmapHex      = fmt.Errorf("invalid bitmap hex")
)

type FieldError struct {
	Field int
	Err   error
}

func (fe *FieldError) Error() string {
	return fmt.Sprintf("field %d: %v", fe.Field, fe.Err)
}

type ValidationError struct {
	Field   int
	Rule    string
	Message string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %d (%s): %s", ve.Field, ve.Rule, ve.Message)
}

type TLVError struct {
	Tag []byte
	Err error
}

func (te *TLVError) Error() string {
	return fmt.Sprintf("TLV tag %x: %v", te.Tag, te.Err)
}
