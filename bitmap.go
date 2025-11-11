package iso8583

import (
	"encoding/hex"
	"fmt"
)

// BitmapManager handles operations for the ISO8583 64-bit primary
// and 64-bit secondary bitmaps.
type BitmapManager struct {
	primary      [BitmapSize]byte          // 8 bytes for 64 bits
	secondary    [SecondaryBitmapSize]byte // 8 bytes for 64 bits
	hasSecondary bool                      // True if DE 1 is set
}

// NewBitmapManager creates a new bitmap manager.
func NewBitmapManager() *BitmapManager {
	return &BitmapManager{}
}

// SetField sets the bit for the given field number (1-128).
// It automatically sets the secondary bitmap indicator (DE 1) if fieldNum > 64.
func (bm *BitmapManager) SetField(fieldNum int) error {
	if fieldNum < 1 || fieldNum > MaxFieldNumber {
		return fmt.Errorf("field number %d out of range", fieldNum)
	}

	if fieldNum <= 64 {
		// Primary bitmap
		byteIndex := (fieldNum - 1) / 8
		bitIndex := 7 - ((fieldNum - 1) % 8) // Bits are 7 (MSB) to 0 (LSB)
		bm.primary[byteIndex] |= (1 << bitIndex)

		// Note: Field 1 is the secondary bitmap indicator.
		// If fieldNum > 64, this logic will be hit by the 'else' block.
		// If fieldNum == 1, it will be set here, which is correct.
	} else {
		// Secondary bitmap
		bm.hasSecondary = true
		bm.primary[0] |= 0x80 // Set bit 1 (MSB of first byte)

		adjustedField := fieldNum - 64
		byteIndex := (adjustedField - 1) / 8
		bitIndex := 7 - ((adjustedField - 1) % 8)
		bm.secondary[byteIndex] |= (1 << bitIndex)
	}

	return nil
}

// IsFieldSet checks if the bit for the given field number is set.
func (bm *BitmapManager) IsFieldSet(fieldNum int) bool {
	if fieldNum < 1 || fieldNum > MaxFieldNumber {
		return false
	}

	if fieldNum <= 64 {
		// Primary bitmap
		byteIndex := (fieldNum - 1) / 8
		bitIndex := 7 - ((fieldNum - 1) % 8)
		return (bm.primary[byteIndex] & (1 << bitIndex)) != 0
	} else {
		// Secondary bitmap
		if !bm.hasSecondary {
			return false // No secondary bitmap, so field can't be set
		}

		adjustedField := fieldNum - 64
		byteIndex := (adjustedField - 1) / 8
		bitIndex := 7 - ((adjustedField - 1) % 8)
		return (bm.secondary[byteIndex] & (1 << bitIndex)) != 0
	}
}

// ClearField clears the bit for the given field number.
// If clearing the last field in the secondary bitmap, it also clears DE 1.
func (bm *BitmapManager) ClearField(fieldNum int) error {
	if fieldNum < 1 || fieldNum > MaxFieldNumber {
		return fmt.Errorf("field number %d out of range", fieldNum)
	}

	if fieldNum <= 64 {
		// Primary bitmap
		byteIndex := (fieldNum - 1) / 8
		bitIndex := 7 - ((fieldNum - 1) % 8)
		bm.primary[byteIndex] &^= (1 << bitIndex) // &^= is (AND NOT)
	} else {
		// Secondary bitmap
		if !bm.hasSecondary {
			return nil // Already clear
		}

		adjustedField := fieldNum - 64
		byteIndex := (adjustedField - 1) / 8
		bitIndex := 7 - ((adjustedField - 1) % 8)
		bm.secondary[byteIndex] &^= (1 << bitIndex)

		// Check if secondary bitmap is now empty
		isEmpty := true
		for i := 0; i < SecondaryBitmapSize; i++ {
			if bm.secondary[i] != 0 {
				isEmpty = false
				break
			}
		}

		// If empty, clear the secondary indicator (DE 1)
		if isEmpty {
			bm.hasSecondary = false
			bm.primary[0] &^= 0x80 // Clear bit 1
		}
	}

	return nil
}

// GetPresentFields returns a slice of field numbers that are set in the bitmap.
func (bm *BitmapManager) GetPresentFields() []int {
	fields := make([]int, 0, 64) // Pre-allocate for common case

	// Check primary bitmap (fields 2-64)
	for i := 1; i < BitmapSize*8; i++ { // Start from i=1 (field 2)
		fieldNum := i + 1
		if bm.IsFieldSet(fieldNum) {
			fields = append(fields, fieldNum)
		}
	}

	// Check secondary bitmap if present (fields 65-128)
	if bm.hasSecondary {
		for i := 0; i < SecondaryBitmapSize*8; i++ {
			fieldNum := i + 65
			if bm.IsFieldSet(fieldNum) {
				fields = append(fields, fieldNum)
			}
		}
	}

	return fields
}

// PackBitmap packs the bitmap into the buffer, using the specified encoding.
// Returns the number of bytes written.
func (bm *BitmapManager) PackBitmap(buf []byte, encoding BitmapEncoding) (int, error) {
	if encoding == BitmapEncodingHex {
		return bm.packBitmapHex(buf)
	}
	return bm.packBitmapBinary(buf)
}

// packBitmapBinary packs the bitmap as raw binary bytes (8 or 16 bytes).
func (bm *BitmapManager) packBitmapBinary(buf []byte) (int, error) {
	offset := 0
	totalSize := BitmapSize // 8 bytes
	if bm.hasSecondary {
		totalSize += SecondaryBitmapSize // +8 = 16 bytes
	}
	if len(buf) < totalSize {
		return 0, ErrBufferTooSmall
	}

	// Write primary bitmap
	copy(buf[offset:], bm.primary[:])
	offset += BitmapSize

	// Write secondary bitmap if present
	if bm.hasSecondary {
		copy(buf[offset:], bm.secondary[:])
		offset += SecondaryBitmapSize
	}

	return offset, nil
}

// packBitmapHex packs the bitmap as a hex-encoded string (16 or 32 chars).
func (bm *BitmapManager) packBitmapHex(buf []byte) (int, error) {
	const hexBitmapSize = 16 // 8 bytes * 2 hex chars
	offset := 0

	// Check space for primary bitmap (16 hex chars)
	if len(buf) < offset+hexBitmapSize {
		return 0, ErrBufferTooSmall
	}
	// hex.Encode(buf[offset:offset+hexBitmapSize], bm.primary[:])
	encodeHexUpper(buf[offset:offset+hexBitmapSize], bm.primary[:])
	offset += hexBitmapSize

	// Check space and write secondary bitmap if present
	if bm.hasSecondary {
		if len(buf) < offset+hexBitmapSize {
			return 0, ErrBufferTooSmall
		}
		// hex.Encode(buf[offset:offset+hexBitmapSize], bm.secondary[:])
		encodeHexUpper(buf[offset:offset+hexBitmapSize], bm.secondary[:])
		offset += hexBitmapSize
	}
	return offset, nil
}

// UnpackBitmap unpacks the bitmap from the data buffer, using the specified encoding.
// Returns the number of bytes consumed.
func (bm *BitmapManager) UnpackBitmap(data []byte, encoding BitmapEncoding) (int, error) {
	if encoding == BitmapEncodingHex {
		return bm.unpackBitmapHex(data)
	}
	return bm.unpackBitmapBinary(data)
}

// unpackBitmapBinary unpacks the bitmap from raw binary bytes.
func (bm *BitmapManager) unpackBitmapBinary(data []byte) (int, error) {
	if len(data) < BitmapSize { // Must have at least primary
		return 0, ErrInvalidBitmap
	}

	// Read primary bitmap
	copy(bm.primary[:], data[:BitmapSize])
	offset := BitmapSize

	// Check DE 1 (MSB of first byte)
	bm.hasSecondary = (bm.primary[0] & 0x80) != 0

	if bm.hasSecondary {
		// Must have secondary bitmap
		if len(data) < offset+SecondaryBitmapSize {
			return 0, ErrInvalidBitmap
		}
		copy(bm.secondary[:], data[offset:offset+SecondaryBitmapSize])
		offset += SecondaryBitmapSize
	} else {
		// Ensure secondary bitmap is zeroed out
		for i := range bm.secondary {
			bm.secondary[i] = 0
		}
	}
	return offset, nil
}

// unpackBitmapHex unpacks the bitmap from a hex-encoded string.
func (bm *BitmapManager) unpackBitmapHex(data []byte) (int, error) {
	const hexBitmapSize = 16 // 16 hex chars
	if len(data) < hexBitmapSize {
		return 0, ErrInvalidBitmap
	}

	// Unpack primary bitmap (16 hex chars -> 8 bytes)
	_, err := hex.Decode(bm.primary[:], data[:hexBitmapSize])
	if err != nil {
		return 0, ErrInvalidBitmapHex
	}
	offset := hexBitmapSize

	// Check DE 1
	bm.hasSecondary = (bm.primary[0] & 0x80) != 0

	if bm.hasSecondary {
		// Must have secondary bitmap (another 16 hex chars)
		if len(data) < offset+hexBitmapSize {
			return 0, ErrInvalidBitmap
		}
		_, err := hex.Decode(bm.secondary[:], data[offset:offset+hexBitmapSize])
		if err != nil {
			return 0, ErrInvalidBitmapHex
		}
		offset += hexBitmapSize
	} else {
		// Ensure secondary bitmap is zeroed out
		for i := range bm.secondary {
			bm.secondary[i] = 0
		}
	}
	return offset, nil
}

// Reset clears all bits in both bitmaps.
func (bm *BitmapManager) Reset() {
	for i := range bm.primary {
		bm.primary[i] = 0
	}
	for i := range bm.secondary {
		bm.secondary[i] = 0
	}
	bm.hasSecondary = false
}

// HasSecondaryBitmap returns true if the secondary bitmap indicator (DE 1) is set.
func (bm *BitmapManager) HasSecondaryBitmap() bool {
	return bm.hasSecondary
}

// BitmapSize returns the total size of the bitmap in bytes (8 or 16).
func (bm *BitmapManager) BitmapSize() int {
	size := BitmapSize
	if bm.hasSecondary {
		size += SecondaryBitmapSize
	}
	return size
}
