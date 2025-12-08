package helpers

import "strings"

func IsNotExistErr(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found"))
}

func IsForbiddenErr(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "forbidden"))
}
