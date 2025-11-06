package iso8583

var DefaultConfigField = map[int]FieldConfig{
	// Field 1 is the Bitmap, handled automatically by the library

	2:  {Type: FieldTypeN, Length: LengthLLVAR, MaxLength: 19, Mandatory: false},     // Primary Account Number (PAN)
	3:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 6, Mandatory: true},       // Processing Code
	4:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 12, Mandatory: true},      // Amount, Transaction
	5:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 12, Mandatory: false},     // Amount, Settlement (User had 'true', common default is 'false')
	6:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 12, Mandatory: false},     // Amount, Cardholder Billing (User had 'true', common default is 'false')
	7:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 10, Mandatory: true},      // Transmission Date & Time (MMDDhhmmss)
	8:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 8, Mandatory: false},      // Amount, Cardholder Billing Fee
	9:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 8, Mandatory: false},      // Conversion Rate, Settlement
	10: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 8, Mandatory: false},      // Conversion Rate, Cardholder Billing
	11: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 6, Mandatory: true},       // System Trace Audit Number (STAN)
	12: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 6, Mandatory: true},       // Time, Local Transaction (hhmmss)
	13: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 4, Mandatory: true},       // Date, Local Transaction (MMDD)
	14: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 4, Mandatory: false},      // Date, Expiration
	15: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 4, Mandatory: false},      // Date, Settlement
	16: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 4, Mandatory: false},      // Date, Conversion
	17: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 4, Mandatory: false},      // Date, Capture
	18: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 4, Mandatory: false},      // Merchant Type
	19: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 4, Mandatory: false},      // Acquiring Institution Country Code
	20: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 4, Mandatory: false},      // PAN Extended, Country Code
	21: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 3, Mandatory: false},      // Forwarding Institution Country Code
	22: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 3, Mandatory: true},       // Point of Service Entry Mode
	23: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 3, Mandatory: false},      // Application PAN Sequence Number
	24: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 3, Mandatory: false},      // Function Code (ISO 8583:1993) / Network International Identifier
	25: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 2, Mandatory: true},       // Point of Service Condition Code
	26: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 2, Mandatory: false},      // Point of Service Capture Code
	27: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 3, Mandatory: false},      // Authorizing Identification Response Length
	28: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 9, Mandatory: false},      // Amount, Transaction Fee (X+N 8)
	29: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 3, Mandatory: false},      // Amount, Settlement Fee (X+N 8)
	30: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 3, Mandatory: false},      // Amount, Transaction Processing Fee (X+N 8)
	31: {Type: FieldTypeN, Length: LengthLLVAR, MaxLength: 99, Mandatory: false},     // Amount, Settlement Processing Fee (X+N 8)
	32: {Type: FieldTypeN, Length: LengthLLVAR, MaxLength: 99, Mandatory: false},     // Acquiring Institution Identification Code
	33: {Type: FieldTypeN, Length: LengthLLVAR, MaxLength: 99, Mandatory: false},     // Forwarding Institution Identification Code
	34: {Type: FieldTypeANS, Length: LengthLLVAR, MaxLength: 28, Mandatory: false},   // Primary Account Number, Extended
	35: {Type: FieldTypeZ, Length: LengthLLVAR, MaxLength: 37, Mandatory: false},     // Track 2 Data
	36: {Type: FieldTypeZ, Length: LengthLLVAR, MaxLength: 99, Mandatory: false},     // Track 3 Data
	37: {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 12, Mandatory: false},   // Retrieval Reference Number
	38: {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 6, Mandatory: false},    // Authorization Identification Response
	39: {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 2, Mandatory: false},    // Response Code
	40: {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 3, Mandatory: false},    // Service Restriction Code
	41: {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 8, Mandatory: false},    // Card Acceptor Terminal Identification
	42: {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 15, Mandatory: false},   // Card Acceptor Identification Code
	43: {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 40, Mandatory: false},   // Card Acceptor Name/Location
	44: {Type: FieldTypeANS, Length: LengthLLVAR, MaxLength: 25, Mandatory: false},   // Additional Response Data
	45: {Type: FieldTypeANS, Length: LengthLLVAR, MaxLength: 76, Mandatory: false},   // Track 1 Data
	46: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Additional Data - ISO
	47: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Additional Data - National
	48: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Additional Data - Private
	49: {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 3, Mandatory: true},     // Currency Code, Transaction
	50: {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 3, Mandatory: false},    // Currency Code, Settlement
	51: {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 3, Mandatory: false},    // Currency Code, Cardholder Billing
	52: {Type: FieldTypeB, Length: LengthFixed, MaxLength: 16, Mandatory: false},     // Personal Identification Number (PIN) Data
	53: {Type: FieldTypeN, Length: LengthFixed, MaxLength: 16, Mandatory: false},     // Security Related Control Information
	54: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 120, Mandatory: false}, // Additional Amounts
	55: {Type: FieldTypeB, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false},   // ICC Data (EMV)
	56: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved ISO
	57: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved National
	58: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved National
	59: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved National
	60: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved Private
	61: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved Private
	62: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved Private
	63: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved Private
	64: {Type: FieldTypeB, Length: LengthFixed, MaxLength: 8, Mandatory: false},      // Message Authentication Code (MAC)

	// --- Secondary Bitmap Fields (65-128) ---

	65:  {Type: FieldTypeB, Length: LengthFixed, MaxLength: 1, Mandatory: false},      // Extended Bitmap
	66:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 1, Mandatory: false},      // Settlement Code
	67:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 2, Mandatory: false},      // Extended Payment Code
	68:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 3, Mandatory: false},      // Receiving Institution Country Code
	69:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 3, Mandatory: false},      // Settlement Institution Country Code
	70:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 3, Mandatory: false},      // Network Management Information Code
	71:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 4, Mandatory: false},      // Message Number
	72:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 4, Mandatory: false},      // Message Number, Last
	73:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 6, Mandatory: false},      // Date, Action (YYYYMMDD)
	74:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 10, Mandatory: false},     // Credits, Number
	75:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 10, Mandatory: false},     // Credits, Reversal Number
	76:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 10, Mandatory: false},     // Debits, Number
	77:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 10, Mandatory: false},     // Debits, Reversal Number
	78:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 10, Mandatory: false},     // Transfer, Number
	79:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 10, Mandatory: false},     // Transfer, Reversal Number
	80:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 10, Mandatory: false},     // Inquiries, Number
	81:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 10, Mandatory: false},     // Authorizations, Number
	82:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 12, Mandatory: false},     // Credits, Processing Fee Amount
	83:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 12, Mandatory: false},     // Credits, Transaction Fee Amount
	84:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 12, Mandatory: false},     // Debits, Processing Fee Amount
	85:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 12, Mandatory: false},     // Debits, Transaction Fee Amount
	86:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 16, Mandatory: false},     // Credits, Amount
	87:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 16, Mandatory: false},     // Credits, Reversal Amount
	88:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 16, Mandatory: false},     // Debits, Amount
	89:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 16, Mandatory: false},     // Debits, Reversal Amount
	90:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 42, Mandatory: false},     // Original Data Elements
	91:  {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 1, Mandatory: false},    // File Update Code
	92:  {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 2, Mandatory: false},    // File Security Code
	93:  {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 5, Mandatory: false},    // Response Indicator
	94:  {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 7, Mandatory: false},    // Service Indicator
	95:  {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 42, Mandatory: false},   // Replacement Amounts
	96:  {Type: FieldTypeB, Length: LengthFixed, MaxLength: 8, Mandatory: false},      // Message Security Code
	97:  {Type: FieldTypeN, Length: LengthFixed, MaxLength: 17, Mandatory: false},     // Amount, Net Settlement (X+N 16)
	98:  {Type: FieldTypeANS, Length: LengthFixed, MaxLength: 25, Mandatory: false},   // Payee
	99:  {Type: FieldTypeN, Length: LengthLLVAR, MaxLength: 11, Mandatory: false},     // Settlement Institution Identification Code
	100: {Type: FieldTypeN, Length: LengthLLVAR, MaxLength: 11, Mandatory: false},     // Receiving Institution Identification Code
	101: {Type: FieldTypeANS, Length: LengthLLVAR, MaxLength: 17, Mandatory: false},   // File Name
	102: {Type: FieldTypeANS, Length: LengthLLVAR, MaxLength: 28, Mandatory: false},   // Account Identification 1
	103: {Type: FieldTypeANS, Length: LengthLLVAR, MaxLength: 28, Mandatory: false},   // Account Identification 2
	104: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 100, Mandatory: false}, // Transaction Description
	105: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for ISO Use
	106: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for ISO Use
	107: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for ISO Use
	108: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for ISO Use
	109: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for ISO Use
	110: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for ISO Use
	111: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for ISO Use
	112: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for National Use
	113: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for National Use
	114: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for National Use
	115: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for National Use
	116: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for National Use
	117: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for National Use
	118: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for National Use
	119: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for National Use
	120: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for Private Use
	121: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for Private Use
	122: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for Private Use
	123: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for Private Use
	124: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for Private Use
	125: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for Private Use
	126: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for Private Use
	127: {Type: FieldTypeANS, Length: LengthLLLVAR, MaxLength: 999, Mandatory: false}, // Reserved for Private Use
	128: {Type: FieldTypeB, Length: LengthFixed, MaxLength: 8, Mandatory: false},      // Message Authentication Code (MAC)
}
