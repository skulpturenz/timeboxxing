package semantic

import (
	"encoding/binary"
	"fmt"
	"math"
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

func EncodeFloat32Vector(values []float32) ([]byte, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}
	data := make([]byte, len(values)*4)
	for index, value := range values {
		binary.LittleEndian.PutUint32(data[index*4:], math.Float32bits(value))
	}
	return data, nil
}
