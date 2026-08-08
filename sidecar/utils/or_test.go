package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOr_ReturnsTheFirstMatch(t *testing.T) {
	t.Parallel()

	longerThanOne := func(x string) bool {
		return len(x) > 1
	}

	result := Or(longerThanOne, "a", "bb", "ccc")

	require.NotNil(t, result)
	assert.Equal(t, "bb", *result, "the first match wins, not the best one")
}

func TestOr_ReturnsNilWhenNothingMatches(t *testing.T) {
	t.Parallel()

	never := func(_ string) bool {
		return false
	}

	assert.Nil(t, Or(never, "a", "b"), "no match is an absent value")
}

func TestOr_ReturnsNilForNoItems(t *testing.T) {
	t.Parallel()

	always := func(_ string) bool {
		return true
	}

	assert.Nil(t, Or(always), "nothing to pick from is an absent value")
}

func TestOr_PointsAtACopyOfTheMatch(t *testing.T) {
	t.Parallel()

	always := func(_ string) bool {
		return true
	}

	items := []string{"first", "second"}

	result := Or(always, items...)
	require.NotNil(t, result)

	*result = "changed"

	assert.Equal(t, []string{"first", "second"}, items, "the pointer is to the loop copy, never the slice")
}

func TestIsEmptyString_TreatsWhitespaceOnlyAsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		empty bool
	}{
		{name: "the empty string", value: "", empty: true},
		{name: "spaces", value: "   ", empty: true},
		{name: "tabs and newlines", value: "\t\n", empty: true},
		{name: "a word", value: "a", empty: false},
		{name: "a padded word", value: " a ", empty: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.empty, IsEmptyString(test.value))
			assert.Equal(t, test.empty, IsEmptyString(&test.value), "the pointer form trims the same way")
		})
	}
}

func TestIsEmptyString_TreatsANilPointerAsEmpty(t *testing.T) {
	t.Parallel()

	var value *string

	assert.True(t, IsEmptyString(value), "an absent string is an empty one")
}

func TestIsZero_ReportsTheZeroValueOfComparableTypes(t *testing.T) {
	t.Parallel()

	type point struct {
		x int
		y int
	}

	assert.True(t, IsZero(0))
	assert.True(t, IsZero(""))
	assert.True(t, IsZero(point{x: 0, y: 0}))

	assert.False(t, IsZero(1))
	assert.False(t, IsZero("a"))
	assert.False(t, IsZero(point{x: 0, y: 1}))
}
