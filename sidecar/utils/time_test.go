package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Between: inclusive on both bounds ------------------------------------------------------

func TestTimeSpan_BetweenIsInclusiveOnBothBounds(t *testing.T) {
	t.Parallel()

	window := span(0, 10)

	assert.True(t, span(0, 10).Between(window), "a span equal to the window is inside it")
	assert.True(t, span(2, 5).Between(window), "a contained span is inside it")
	assert.True(t, span(0, 0).Between(window), "a zero-length span on the opening bound is inside it")
	assert.True(t, span(10, 10).Between(window), "a zero-length span on the closing bound is inside it")
}

func TestTimeSpan_BetweenRejectsSpansCrossingABound(t *testing.T) {
	t.Parallel()

	window := span(0, 10)

	assert.False(t, span(-1, 5).Between(window), "starting before the window puts the span outside it")
	assert.False(t, span(5, 11).Between(window), "ending after the window puts the span outside it")
}

// --- Overlaps: strict and half-open ---------------------------------------------------------

func TestTimeSpan_OverlapsIsStrictAtTheBounds(t *testing.T) {
	t.Parallel()

	window := span(0, 10)

	assert.False(t, span(-5, 0).Overlaps(window), "ending exactly on the opening bound is only touching")
	assert.False(t, span(10, 15).Overlaps(window), "starting exactly on the closing bound is only touching")
	assert.False(t, span(5, 5).Overlaps(span(5, 5)), "a zero-length span does not even overlap itself")
}

func TestTimeSpan_OverlapsPartialAndContainedSpans(t *testing.T) {
	t.Parallel()

	window := span(0, 10)

	assert.True(t, span(-5, 5).Overlaps(window), "a span reaching into the window overlaps it")
	assert.True(t, span(5, 15).Overlaps(window), "a span reaching out of the window overlaps it")
	assert.True(t, span(2, 5).Overlaps(window), "a contained span overlaps the window")
	assert.True(t, span(-5, 15).Overlaps(window), "a span covering the window overlaps it")
}

// --- Clip ------------------------------------------------------------------------------------

func TestTimeSpan_ClipNarrowsToTheWindow(t *testing.T) {
	t.Parallel()

	window := span(0, 10)

	clipped, ok := span(-5, 15).Clip(window)
	require.True(t, ok)
	assert.Equal(t, window, clipped, "both ends are pulled back to the window")

	clipped, ok = span(5, 15).Clip(window)
	require.True(t, ok)
	assert.Equal(t, span(5, 10), clipped, "only the end outside the window moves")
}

func TestTimeSpan_ClipReportsFalseWhenNothingFallsInside(t *testing.T) {
	t.Parallel()

	window := span(0, 10)

	clipped, ok := span(20, 30).Clip(window)
	assert.False(t, ok, "a disjoint span clips to nothing")
	assert.Equal(t, TimeSpan{}, clipped, "the failed clip returns the zero span, not a partly clipped one")

	clipped, ok = span(10, 20).Clip(window)
	assert.False(t, ok, "a span that only touches the closing bound clips to nothing")
	assert.Equal(t, TimeSpan{}, clipped)
}

func TestTimeSpan_ClipLeavesTheReceiverUnchanged(t *testing.T) {
	t.Parallel()

	original := span(-5, 15)

	_, ok := original.Clip(span(0, 10))

	require.True(t, ok)
	assert.Equal(t, span(-5, 15), original, "TimeSpan is an array behind a value receiver, so Clip narrows a copy")
}

// --- Duration ----------------------------------------------------------------------------------

func TestTimeSpan_DurationIsTheLengthOfTheSpan(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 10*time.Minute, span(0, 10).Duration())
}

func TestTimeSpan_DurationClampsZeroLengthAndInvertedSpansToZero(t *testing.T) {
	t.Parallel()

	assert.Zero(t, span(5, 5).Duration(), "a zero-length span has no duration")
	assert.Zero(t, span(10, 0).Duration(), "an inverted span is clamped to zero rather than reported as negative")
}

// --- ClippedDuration -----------------------------------------------------------------------------

func TestTimeSpan_ClippedDurationMeasuresOnlyThePartInsideTheWindow(t *testing.T) {
	t.Parallel()

	window := span(0, 10)

	assert.Equal(t, 10*time.Minute, span(-5, 15).ClippedDuration(window), "the part outside the window is not counted")
	assert.Equal(t, 5*time.Minute, span(5, 20).ClippedDuration(window))
}

func TestTimeSpan_ClippedDurationIsZeroWhenTheSpanFallsOutside(t *testing.T) {
	t.Parallel()

	window := span(0, 10)

	assert.Zero(t, span(20, 30).ClippedDuration(window), "a disjoint span contributes nothing")
	assert.Zero(t, span(10, 20).ClippedDuration(window), "a touching span contributes nothing")
}
