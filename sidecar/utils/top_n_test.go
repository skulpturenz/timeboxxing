package utils

import (
	"cmp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ascending is a total order: SortFunc is unstable and its input arrives in map iteration order, so
// a comparator that reports ties would make the selected set genuinely nondeterministic.
func ascending(a int, b int) int {
	return cmp.Compare(a, b)
}

func TestTopN_KeepsTheTopSliceInOriginalOrderWithItsDuplicates(t *testing.T) {
	t.Parallel()

	result := TopN(ascending, 50)([]int{1, 5, 3, 5, 2})

	assert.Equal(t, []int{5, 3, 5}, []int(result),
		"TopN filters rather than ranks: input order and multiplicity survive")
}

func TestTopN_ComputesTheCutOffOverDistinctValues(t *testing.T) {
	t.Parallel()

	result := TopN(ascending, 50)([]int{5, 5, 5, 5, 1, 2})

	assert.Equal(t, []int{5, 5, 5, 5, 2}, []int(result),
		"the cut-off is half of the three distinct values, not half of the six items")
}

func TestTopN_RoundsTheCutOffUp(t *testing.T) {
	t.Parallel()

	result := TopN(ascending, 50)([]int{1, 2, 3})

	assert.Equal(t, []int{2, 3}, []int(result), "half of three distinct values rounds up to two")
}

func TestTopN_SelectsNothingAtZeroPercent(t *testing.T) {
	t.Parallel()

	assert.Empty(t, TopN(ascending, 0)([]int{1, 2, 3}))
}

func TestTopN_KeepsEverythingAtAHundredPercent(t *testing.T) {
	t.Parallel()

	result := TopN(ascending, percentMax)([]int{1, 3, 2, 3})

	assert.Equal(t, []int{1, 3, 2, 3}, []int(result), "duplicates come back too")
}

func TestTopN_ReturnsAnEmptyResultForAnEmptyInput(t *testing.T) {
	t.Parallel()

	assert.Empty(t, TopN(ascending, 50)([]int{}))
}

func TestTopN_ReturnsAReusableFilter(t *testing.T) {
	t.Parallel()

	top := TopN(ascending, 50)

	assert.Equal(t, []int{3, 4}, []int(top([]int{1, 2, 3, 4})))
	assert.Equal(t, []int{30, 40}, []int(top([]int{10, 20, 30, 40})), "the filter holds no state between calls")
}

func TestResultTopN_DistinctDedupesTheSelection(t *testing.T) {
	t.Parallel()

	result := TopN(ascending, 50)([]int{5, 5, 3, 3, 1, 2})

	assert.ElementsMatch(t, []int{5, 3}, result.Distinct(),
		"Distinct returns map iteration order, so only membership is meaningful")
}

func TestResultTopN_DistinctIsEmptyForAnEmptySelection(t *testing.T) {
	t.Parallel()

	result := TopN(ascending, 0)([]int{1, 2, 3})

	assert.Empty(t, result.Distinct())
}
