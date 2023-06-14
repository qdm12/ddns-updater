package utils

import (
	"io"
	"strings"
)

func BodyToSingleLine(body io.Reader) (s string) {
	b, err := io.ReadAll(body)
	if err != nil {
		return ""
	}
	data := string(b)
	return ToSingleLine(data)
}

func ToSingleLine(s string) (line string) {
	line = strings.ReplaceAll(s, "\n", "")
	line = strings.ReplaceAll(line, "\r", "")
	line = strings.ReplaceAll(line, "  ", " ")
	line = strings.ReplaceAll(line, "  ", " ")
	return line
}
