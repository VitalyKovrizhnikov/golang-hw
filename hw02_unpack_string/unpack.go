package hw02unpackstring

import (
	"errors"
	"strings"
)

var ErrInvalidString = errors.New("invalid string")

func Unpack(input string) (string, error) {
	runes := []rune(input)
	var builder strings.Builder

	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' {
			i++
			if i >= len(runes) {
				return "", ErrInvalidString
			}
			if runes[i] != '\\' && !isDigit(runes[i]) {
				return "", ErrInvalidString
			}
		} else if isDigit(runes[i]) {
			return "", ErrInvalidString
		}

		char := runes[i]
		count := 1
		if i+1 < len(runes) && isDigit(runes[i+1]) {
			i++
			count = int(runes[i] - '0')
		}

		if count > 0 {
			builder.WriteString(strings.Repeat(string(char), count))
		}
	}

	return builder.String(), nil
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
