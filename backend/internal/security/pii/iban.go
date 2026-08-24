package pii

import (
	"math/big"
	"regexp"
	"strings"
)

var (
	// Turkish IBAN format: TR followed by 24 alphanumeric digits (Total 26 characters)
	trIbanRegex = regexp.MustCompile(`\bTR\d{2}[0-9A-Z]{5}[0-9A-Z]{17}\b`)
)

// ValidateIBAN validates international IBAN with mod-97 checksum (ISO 7064)
func ValidateIBAN(iban string) bool {
	// Clean whitespace and dashes
	cleaned := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(iban, " ", ""), "-", ""))

	if len(cleaned) < 15 || len(cleaned) > 34 {
		return false
	}

	// For Turkey, length must be exactly 26 chars starting with TR
	if strings.HasPrefix(cleaned, "TR") && len(cleaned) != 26 {
		return false
	}

	// Move the first 4 characters (country code + check digits) to the end
	rearranged := cleaned[4:] + cleaned[:4]

	// Convert letters to numbers (A=10, B=11, ..., Z=35)
	var numericStr strings.Builder
	for _, char := range rearranged {
		if char >= '0' && char <= '9' {
			numericStr.WriteRune(char)
		} else if char >= 'A' && char <= 'Z' {
			num := int(char - 'A' + 10)
			numericStr.WriteString(string(rune('0'+num/10)) + string(rune('0'+num%10)))
		} else {
			return false
		}
	}

	// Calculate modulo 97 using big.Int
	n := new(big.Int)
	n, ok := n.SetString(numericStr.String(), 10)
	if !ok {
		return false
	}

	mod := new(big.Int).Mod(n, big.NewInt(97))
	return mod.Int64() == 1
}

// MaskIBAN masks Turkish IBAN: TR12********************34
func MaskIBAN(iban string) string {
	cleaned := strings.ReplaceAll(strings.ReplaceAll(iban, " ", ""), "-", "")
	if len(cleaned) < 8 {
		return "TR********************"
	}
	prefix := cleaned[:4]
	suffix := cleaned[len(cleaned)-2:]
	return prefix + strings.Repeat("*", len(cleaned)-6) + suffix
}
