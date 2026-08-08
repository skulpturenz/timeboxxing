package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoalesce_ReturnsTheFallbackForANilPointer(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "fallback", Coalesce(nil, "fallback"), "a nil pointer is an absent value")
}

func TestCoalesce_ReturnsThePointee(t *testing.T) {
	t.Parallel()

	value := "present"

	assert.Equal(t, "present", Coalesce(&value, "fallback"), "a non-nil pointer wins over the fallback")
}

func TestCoalesce_TreatsAPointerToTheZeroValueAsPresent(t *testing.T) {
	t.Parallel()

	zero := 0

	assert.Equal(t, 0, Coalesce(&zero, 42), "nil means absent; a pointer to the zero value is a present zero")
}
