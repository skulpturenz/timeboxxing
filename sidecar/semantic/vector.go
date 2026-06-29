package semantic

import (
	"encoding/json"
	"fmt"
)

const StoreEmbeddingDimension = 4096

func NormalizeFloat32Vector(values []float32, dimension int) ([]float32, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}
	if dimension <= 0 {
		return nil, fmt.Errorf("embedding dimension must be positive")
	}
	if len(values) > dimension {
		return nil, fmt.Errorf("embedding vector has %d dimensions, exceeds store dimension %d", len(values), dimension)
	}
	if len(values) == dimension {
		return values, nil
	}

	normalized := make([]float32, dimension)
	copy(normalized, values)
	return normalized, nil
}

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
