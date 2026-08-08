package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZeroNil_ReturnsNilForZeroValues(t *testing.T) {
	t.Parallel()

	assert.Nil(t, ZeroNil(""), "the empty string is absent")
	assert.Nil(t, ZeroNil(0), "zero is absent")
	assert.Nil(t, ZeroNil(false), "false is absent")
}

func TestZeroNil_ReturnsAPointerForNonZeroValues(t *testing.T) {
	t.Parallel()

	result := ZeroNil("present")

	require.NotNil(t, result)
	assert.Equal(t, "present", *result)
}

func TestZeroNil_PointsAtACopyOfTheArgument(t *testing.T) {
	t.Parallel()

	value := "present"
	result := ZeroNil(value)
	value = "changed"

	require.NotNil(t, result)
	assert.Equal(t, "present", *result)
	assert.NotEqual(t, value, *result, "the pointer is to the parameter copy, never to the caller's variable")
}
