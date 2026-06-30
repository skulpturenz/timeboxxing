package timesheets

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func randomID(prefix string) (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("read random id: %w", err)
	}
	return prefix + "-" + hex.EncodeToString(raw[:]), nil
}
