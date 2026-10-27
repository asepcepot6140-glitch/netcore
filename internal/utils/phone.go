package utils

import "strings"

const (
	minPhoneDigits = 10
	maxPhoneDigits = 15 // batas E.164
)

// NormalizePhone mengubah nomor WhatsApp ke format digit internasional tanpa "+",
// misalnya "0812-3456-7890" atau "+62 812 3456 7890" menjadi "6281234567890".
// Mengembalikan string kosong jika nomor tidak valid.
func NormalizePhone(raw string) string {
	var digits strings.Builder

	for i, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' && i == 0:
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
		default:
			return ""
		}
	}

	phone := digits.String()

	switch {
	case strings.HasPrefix(phone, "00"):
		phone = phone[2:]
	case strings.HasPrefix(phone, "0"):
		phone = "62" + phone[1:]
	}

	if len(phone) < minPhoneDigits || len(phone) > maxPhoneDigits {
		return ""
	}

	return phone
}
