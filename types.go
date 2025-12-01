package iso8583

import (
	"encoding/json"
	"strings"
	"sync"
)

type FieldType int

const (
	FieldTypeANS FieldType = iota
	FieldTypeAN
	FieldTypeN
	FieldTypeB
	FieldTypeZ
	FieldTypeCustom
)

type BitmapEncoding int

const (
	BitmapEncodingBinary BitmapEncoding = iota
	BitmapEncodingHex
)

type LengthType int

const (
	LengthFixed LengthType = iota
	LengthLLVAR
	LengthLLLVAR
	LengthLLLLVAR
)

type LengthIndicatorType int

const (
	LengthIndicatorNone LengthIndicatorType = iota
	LengthIndicatorBinary
	LengthIndicatorASCII
	LengthIndicatorHex
)

type HeaderType int

const (
	HeaderNone HeaderType = iota
	HeaderBinary
	HeaderASCII
	HeaderHex
	HeaderCustom
)

type TLVType int

const (
	TLVStandard TLVType = iota
	TLVEMV
	TLVASCII
)

type ValidationLevel int

const (
	ValidationNone ValidationLevel = iota
	ValidationBasic
	ValidationStrict
	ValidationCustom
)

type Field struct {
	data      []byte
	length    int
	fieldType FieldType
	parsed    bool
	mu        sync.RWMutex
}

type TLV struct {
	Tag    []byte
	Length int
	Value  []byte
}

type FieldConfig struct {
	Type      FieldType  `json:"type"`
	Length    LengthType `json:"length"`
	MaxLength int        `json:"max_length"`
	MinLength int        `json:"min_length"`
	Mandatory bool       `json:"mandatory"`
	Format    string     `json:"format,omitempty"`
}

func (fc *FieldConfig) UnmarshalJSON(data []byte) error {
	type Alias FieldConfig
	aux := &struct {
		Type interface{} `json:"type"`
		*Alias
	}{
		Alias: (*Alias)(fc),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch v := aux.Type.(type) {
	case float64:
		fc.Type = FieldType(v)
	case string:
		fc.Type = parseFieldTypeString(v)
	}

	return nil
}

func parseFieldTypeString(s string) FieldType {
	switch strings.ToUpper(s) {
	case "ANS":
		return FieldTypeANS
	case "AN":
		return FieldTypeAN
	case "N":
		return FieldTypeN
	case "B":
		return FieldTypeB
	case "Z":
		return FieldTypeZ
	default:
		return FieldTypeCustom
	}
}

type LengthIndicatorConfig struct {
	Type   LengthIndicatorType `json:"type"`
	Length int                 `json:"length"`
}

type HeaderConfig struct {
	Type   HeaderType `json:"type"`
	Length int        `json:"length"`
	Format string     `json:"format,omitempty"`
}

type TLVConfig struct {
	Type     TLVType `json:"type"`
	Enabled  bool    `json:"enabled"`
	MaxDepth int     `json:"max_depth"`
}

type PackagerConfig struct {
	Fields          map[int]FieldConfig   `json:"fields"`
	BitmapEncoding  BitmapEncoding        `json:"bitmap_encoding"`
	LengthIndicator LengthIndicatorConfig `json:"length_indicator"`
	Header          HeaderConfig          `json:"header"`
	TLV             TLVConfig             `json:"tlv"`
}

const (
	DefaultBufferSize   = 8192
	MaxFieldNumber      = 128
	BitmapSize          = 8
	SecondaryBitmapSize = 8
)
