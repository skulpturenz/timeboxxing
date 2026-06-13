package semantic

import (
	"encoding/json"
	"fmt"
)

func EncodeFloat32Vector(values []float32) (string, error) {
	if len(values) == 0 {
		return "", fmt.Errorf("embedding vector is empty")
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode embedding vector: %w", err)
	}
	return string(data), nil
}
