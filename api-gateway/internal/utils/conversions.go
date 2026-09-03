package utils

import "strconv"

// Converts string to int with pre-defined bit size
func ConvertStringToInt(s string, base int, bitSize int) (int, error) {
	converted, err := strconv.ParseInt(s, base, bitSize)
	if err != nil {
		return 0, err
	}
	return int(converted), nil
}
