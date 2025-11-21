package autochecks

import (
	"bytes"
	"os"
)

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func readAll(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func ensureAbsent(path string) {
	_ = os.RemoveAll(path)
}

func containsSuffix(b []byte, suf string) bool {
	return bytes.HasSuffix(b, []byte(suf))
}

func truncateFile(path string, size int64) error {
	return os.Truncate(path, size)
}

func appendString(path, s string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(s))
	_ = f.Close()
	return err
}
