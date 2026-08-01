package enumscategories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Parse must be the exact inverse of String across the whole taxonomy: the codes seeded into
// application_categories are String's output, and a read resolves a persisted classification by
// parsing one back. A code String emits but Parse rejects makes that category unreadable — which is
// what "productivity" and "games" were before.
func TestParse_RoundTripsEveryCategory(t *testing.T) {
	for category := CategoryUnknown; category <= CategoryOther; category++ {
		code := category.String()

		parsed, err := Parse(code)
		require.NoError(t, err, "parse %q", code)
		assert.Equal(t, category, parsed, "code %q must resolve back to the category that emits it", code)
	}
}

// Codes are stored, not typed, so the taxonomy tolerates the whitespace and casing a hand-edited
// row carries.
func TestParse_IgnoresCasingAndSurroundingWhitespace(t *testing.T) {
	parsed, err := Parse("  Graphics-Design ")
	require.NoError(t, err)
	assert.Equal(t, CategoryGraphicsDesign, parsed)
}

// An unrecognized code is a taxonomy this binary does not know, so it is an error rather than a
// silent CategoryUnknown.
func TestParse_RejectsAnUnknownCode(t *testing.T) {
	parsed, err := Parse("spreadsheets")
	require.Error(t, err)
	assert.Equal(t, CategoryUnknown, parsed)
}
